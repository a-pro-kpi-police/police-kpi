package main

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"
	"testing"
)

func TestGlobalKPIAverageAcrossPeriodsAndAccessScopes(t *testing.T) {
	app := newSupportTestApp(t)
	defer app.db.Close()
	ctx := context.Background()
	addDistrict := func(name string) int {
		var id int
		if err := app.db.QueryRowContext(ctx, `INSERT INTO districts(name, coeff) VALUES (?, 1) RETURNING id`, name).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	first := addDistrict("First")
	second := addDistrict("Second")
	empty := addDistrict("Empty")
	head := insertSupportUser(t, app, "average.head", "Head", "Rank", roleHead, &first)
	admin, err := app.getUserByUsername(ctx, "admin")
	if err != nil {
		t.Fatal(err)
	}
	headToken, adminToken := supportToken(t, app, head), supportToken(t, app, admin)
	fixture := scoringFixture{db: app.db}
	catalog := fixture.addCatalog(t, "average", 1, 0, false, false)
	ids := []int{}
	for i := 0; i < 40; i++ {
		district, score := first, 100.0
		if i >= 10 {
			district, score = second, 60
		}
		var id int
		err := app.db.QueryRowContext(ctx, `INSERT INTO users(username, password_hash, full_name, rank, role, district_id)
			VALUES (?, 'test', ?, 'Rank', 'officer', ?) RETURNING id`, fmt.Sprintf("average.%d", i), fmt.Sprintf("Officer %d", i), district).Scan(&id)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
		for part := 0; part < 2; part++ {
			_, err = app.db.ExecContext(ctx, `INSERT INTO kpi_entries(date, month, officer_id, catalog_id, quantity, points_per_unit, final_score, status)
				VALUES ('2026-07-15', '2026-07', ?, ?, 1, 1, ?, 'confirmed')`, id, catalog, score/2)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, row := range []struct {
		month, status   string
		quantity, score float64
	}{
		{"2026-08", "confirmed", 1, 200},
		{"2026-07", "pending", 1, 999},
		{"2026-07", "rejected", 1, 999},
		{"2026-07", "confirmed", 0, 999},
	} {
		_, err := app.db.ExecContext(ctx, `INSERT INTO kpi_entries(date, month, officer_id, catalog_id, quantity, points_per_unit, final_score, status)
			VALUES ('2026-07-15', ?, ?, ?, ?, 1, ?, ?)`, row.month, ids[0], catalog, row.quantity, row.score, row.status)
		if err != nil {
			t.Fatal(err)
		}
	}
	handler := app.routes()
	for _, period := range []struct {
		month          string
		average, score float64
	}{
		{"2026-07", 70, 100}, {"2026-08", 5, 200}, {"", 75, 300}, {"2026-09", 0, 0},
	} {
		for _, scope := range []struct {
			path, token string
			count       int
		}{
			{"/api/rating?", headToken, 10},
			{"/api/admin/rating?", adminToken, 40},
			{fmt.Sprintf("/api/admin/rating?district_id=%d&", first), adminToken, 10},
			{fmt.Sprintf("/api/admin/rating?district_id=%d&", second), adminToken, 30},
			{fmt.Sprintf("/api/admin/rating?district_id=%d&", empty), adminToken, 0},
		} {
			response := supportJSONRequest(t, handler, http.MethodGet, scope.path+"month="+period.month, scope.token, nil)
			if response.Code != http.StatusOK {
				t.Fatalf("%s: %d %s", scope.path, response.Code, response.Body.String())
			}
			var rows []RatingOfficerOut
			decodeSupportResponse(t, response, &rows)
			if len(rows) != scope.count {
				t.Fatalf("%s: count %d, want %d", scope.path, len(rows), scope.count)
			}
			for _, row := range rows {
				if row.GlobalAverageScore != period.average {
					t.Fatalf("%s %s: average %v, want %v", scope.path, period.month, row.GlobalAverageScore, period.average)
				}
				if scope.token == headToken && row.DistrictID != first {
					t.Fatal("rating disclosed another district")
				}
			}
		}
		for _, suffix := range []string{"", fmt.Sprintf("&district_id=%d", first), fmt.Sprintf("&officer_id=%d", ids[0]), fmt.Sprintf("&officer_id=%d", ids[10])} {
			response := supportJSONRequest(t, handler, http.MethodGet, "/api/dashboard?month="+period.month+suffix, headToken, nil)
			if response.Code != http.StatusOK {
				t.Fatalf("dashboard: %d %s", response.Code, response.Body.String())
			}
			var report struct {
				Average  float64            `json:"global_average_score"`
				Officers []RatingOfficerOut `json:"officers"`
			}
			decodeSupportResponse(t, response, &report)
			if report.Average != period.average {
				t.Fatalf("dashboard average %v, want %v", report.Average, period.average)
			}
			for _, row := range report.Officers {
				if row.OfficerID >= ids[10] {
					t.Fatal("dashboard disclosed another district")
				}
				if row.OfficerID == ids[0] && row.Final != period.score {
					t.Fatalf("period score = %v, want %v", row.Final, period.score)
				}
			}
			if strings.Contains(suffix, fmt.Sprintf("officer_id=%d", ids[10])) && len(report.Officers) != 0 {
				t.Fatal("foreign employee must not be visible")
			}
		}
	}
	// An additional employee without entries still contributes to the denominator.
	insertSupportUser(t, app, "average.zero", "Zero", "Rank", roleOfficer, nil)
	average, err := app.globalKPIAverage(ctx, "2026-07")
	if err != nil || math.Abs(average-2800.0/41) > 1e-9 {
		t.Fatalf("zero employee: %v, %v", average, err)
	}
	response := supportJSONRequest(t, handler, http.MethodGet, fmt.Sprintf("/api/dashboard?district_id=%d", second), headToken, nil)
	if response.Code != http.StatusForbidden {
		t.Fatalf("foreign district status = %d", response.Code)
	}
}

func TestGlobalKPIAverageEmptySystem(t *testing.T) {
	app := newSupportTestApp(t)
	defer app.db.Close()
	average, err := app.globalKPIAverage(context.Background(), "")
	if err != nil || average != 0 {
		t.Fatalf("empty average: %v, %v", average, err)
	}
}
