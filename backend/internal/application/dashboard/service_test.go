package dashboard

import (
	"context"
	"reflect"
	"testing"

	domaindashboard "kpi-police/backend/internal/domain/dashboard"
	domainusers "kpi-police/backend/internal/domain/users"
)

// The repository fake applies the same period/officer/district scope requested
// by the service, and records it so dropping a filter fails the test.
type averageRepository struct {
	entries   []domaindashboard.Entry
	officers  []domainusers.User
	filters   []domaindashboard.Filter
	districts []int
}

func (r *averageRepository) ListEntries(_ context.Context, filter domaindashboard.Filter) ([]domaindashboard.Entry, error) {
	r.filters = append(r.filters, filter)
	var result []domaindashboard.Entry
	for _, entry := range r.entries {
		if filter.Month != "" && entry.Month != filter.Month {
			continue
		}
		if filter.OfficerID != nil && entry.OfficerID != *filter.OfficerID {
			continue
		}
		if filter.DistrictID != nil && *r.officers[entry.OfficerID-1].DistrictID != *filter.DistrictID {
			continue
		}
		result = append(result, entry)
	}
	return result, nil
}

func (r *averageRepository) ListCatalogBlocks(context.Context) ([]string, error) { return nil, nil }

func (r *averageRepository) ListOfficers(_ context.Context, districtID int) ([]domainusers.User, error) {
	r.districts = append(r.districts, districtID)
	var result []domainusers.User
	for _, officer := range r.officers {
		if districtID == 0 || *officer.DistrictID == districtID {
			result = append(result, officer)
		}
	}
	return result, nil
}

func TestDashboardAverageWeightsEmployeesAndPreservesScope(t *testing.T) {
	first, second, empty, selected := 1, 2, 3, 11
	for _, tc := range []struct {
		name   string
		filter domaindashboard.Filter
		want   float64
		count  int
	}{
		{"unequal companies", domaindashboard.Filter{Month: "2026-07"}, 70, 40},
		{"first company", domaindashboard.Filter{Month: "2026-07", DistrictID: &first}, 100, 10},
		{"second company", domaindashboard.Filter{Month: "2026-07", DistrictID: &second}, 60, 30},
		{"other month includes inactive employees", domaindashboard.Filter{Month: "2026-08"}, 5, 40},
		{"all months", domaindashboard.Filter{}, 75, 40},
		{"empty company", domaindashboard.Filter{Month: "2026-07", DistrictID: &empty}, 0, 0},
		{"empty month", domaindashboard.Filter{Month: "2026-09"}, 0, 40},
		{"selected employee", domaindashboard.Filter{Month: "2026-07", DistrictID: &second, OfficerID: &selected}, 60, 30},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &averageRepository{}
			for id := 1; id <= 40; id++ {
				district, score := first, 100.0
				if id > 10 {
					district, score = second, 60
				}
				repo.officers = append(repo.officers, domainusers.User{ID: id, DistrictID: &district})
				// Split each employee's KPI across entries to detect averaging entries.
				for part := 0; part < 2; part++ {
					repo.entries = append(repo.entries, domaindashboard.Entry{Month: "2026-07", OfficerID: id, Quantity: 1, FinalScore: score / 2, Status: domaindashboard.StatusConfirmed})
				}
			}
			repo.entries = append(repo.entries,
				domaindashboard.Entry{Month: "2026-08", OfficerID: 1, Quantity: 1, FinalScore: 200, Status: domaindashboard.StatusConfirmed},
				domaindashboard.Entry{Month: "2026-07", OfficerID: 1, Quantity: 1, FinalScore: 999, Status: domaindashboard.StatusPending},
				domaindashboard.Entry{Month: "2026-07", OfficerID: 1, Quantity: 1, FinalScore: 999, Status: domaindashboard.StatusRejected},
			)
			report, err := NewService(repo, repo, domaindashboard.NewPolicy()).BuildDashboard(context.Background(), tc.filter)
			if err != nil {
				t.Fatal(err)
			}
			if report.Comparisons.FocusScore != tc.want || report.Comparisons.OfficerCount != tc.count {
				t.Fatalf("average/count = %v/%d, want %v/%d", report.Comparisons.FocusScore, report.Comparisons.OfficerCount, tc.want, tc.count)
			}
			wantFilters := []domaindashboard.Filter{tc.filter}
			if tc.filter.Month != "" || tc.filter.OfficerID != nil {
				wantFilters = append(wantFilters, domaindashboard.Filter{DistrictID: tc.filter.DistrictID})
			}
			if !reflect.DeepEqual(repo.filters, wantFilters) {
				t.Fatalf("entry filters = %#v, want %#v", repo.filters, wantFilters)
			}
			if !reflect.DeepEqual(repo.districts, []int{optionalIDValue(tc.filter.DistrictID)}) {
				t.Fatalf("officer district scope = %v", repo.districts)
			}
		})
	}
}
