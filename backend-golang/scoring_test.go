package main

import (
	"context"
	"database/sql"
	"testing"
)

type scoringFixture struct {
	db        *Store
	officerID int
}

func newScoringFixture(t *testing.T) scoringFixture {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatal(err)
	}
	store := &Store{db: db, dialect: dialectSQLite}
	if err := createSchema(store); err != nil {
		t.Fatal(err)
	}
	var districtID int
	err = store.QueryRowContext(context.Background(), `INSERT INTO districts(name, coeff) VALUES('Test', 1.0) RETURNING id`).Scan(&districtID)
	if err != nil {
		t.Fatal(err)
	}
	var officerID int
	err = store.QueryRowContext(context.Background(), `INSERT INTO users(username, password_hash, full_name, rank, role, district_id, monthly_norm)
		VALUES('officer', 'test', 'Test Officer', 'Inspector', 'officer', ?, 100) RETURNING id`, districtID).Scan(&officerID)
	if err != nil {
		t.Fatal(err)
	}
	return scoringFixture{db: store, officerID: officerID}
}

func (f scoringFixture) addCatalog(t *testing.T, code string, points, limit float64, isMass, isCritical bool) int {
	t.Helper()
	var id int
	err := f.db.QueryRowContext(context.Background(), `INSERT INTO kpi_catalog
		(code, indicator, block, points_per_unit, unit, monthly_limit, is_mass, is_critical, is_active)
		VALUES (?, ?, 'Test', ?, 'unit', ?, ?, ?, 1)
		RETURNING id`,
		code, code, points, limit, boolInt(isMass), boolInt(isCritical),
	).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func (f scoringFixture) addEntry(t *testing.T, catalogID int, quantity float64) int {
	t.Helper()
	var points float64
	if err := f.db.QueryRowContext(context.Background(), `SELECT points_per_unit FROM kpi_catalog WHERE id = ?`, catalogID).Scan(&points); err != nil {
		t.Fatal(err)
	}
	var id int
	err := f.db.QueryRowContext(context.Background(), `INSERT INTO kpi_entries
		(date, month, officer_id, catalog_id, quantity, points_per_unit, coeff, status)
		VALUES('2026-07-15', '2026-07', ?, ?, ?, ?, 1.0, 'confirmed')
		RETURNING id`,
		f.officerID, catalogID, quantity, points,
	).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestMonthlyLimitsPreserveReportedQuantity(t *testing.T) {
	fixture := newScoringFixture(t)
	defer fixture.db.Close()
	entry1 := fixture.addEntry(t, fixture.addCatalog(t, "A590-1", 1, 20, true, false), 35)
	entry2 := fixture.addEntry(t, fixture.addCatalog(t, "A590-3", 5, 6, false, false), 8)

	result, err := recalculateOfficerMonth(context.Background(), fixture.db, fixture.officerID, "2026-07")
	if err != nil {
		t.Fatal(err)
	}
	assertEntryScores(t, fixture.db, entry1, 35, 20, 20)
	assertEntryScores(t, fixture.db, entry2, 8, 6, 30)
	if result.FinalScore != 50 {
		t.Fatalf("final score = %v, want 50", result.FinalScore)
	}
}

func TestMassFilterUsesCappedPositiveScores(t *testing.T) {
	fixture := newScoringFixture(t)
	defer fixture.db.Close()
	fixture.addEntry(t, fixture.addCatalog(t, "MASS", 1, 30, true, false), 30)
	fixture.addEntry(t, fixture.addCatalog(t, "OTHER", 5, 6, false, false), 6)

	result, err := recalculateOfficerMonth(context.Background(), fixture.db, fixture.officerID, "2026-07")
	if err != nil {
		t.Fatal(err)
	}
	if result.MassShare != 0.5 {
		t.Fatalf("mass share = %v, want 0.5", result.MassShare)
	}
	if result.AntiCoeff != 0.85 {
		t.Fatalf("anti coeff = %v, want 0.85", result.AntiCoeff)
	}
	if result.FinalScore != 51 {
		t.Fatalf("final score = %v, want 51", result.FinalScore)
	}
}

func TestCriticalIndicatorZeroesMonth(t *testing.T) {
	fixture := newScoringFixture(t)
	defer fixture.db.Close()
	first := fixture.addEntry(t, fixture.addCatalog(t, "ORDINARY", 10, 10, false, false), 5)
	second := fixture.addEntry(t, fixture.addCatalog(t, "CRITICAL", -100, 1, false, true), 1)

	result, err := recalculateOfficerMonth(context.Background(), fixture.db, fixture.officerID, "2026-07")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Critical {
		t.Fatal("critical = false, want true")
	}
	assertFinalScore(t, fixture.db, first, 0)
	assertFinalScore(t, fixture.db, second, 0)
	if result.FinalScore != 0 {
		t.Fatalf("final score = %v, want 0", result.FinalScore)
	}
}

func assertEntryScores(t *testing.T, db *Store, entryID int, quantity, counted, capped float64) {
	t.Helper()
	var gotQuantity, gotCounted, gotCapped float64
	if err := db.QueryRowContext(context.Background(), `SELECT quantity, counted_quantity, capped_score FROM kpi_entries WHERE id = ?`, entryID).Scan(&gotQuantity, &gotCounted, &gotCapped); err != nil {
		t.Fatal(err)
	}
	if gotQuantity != quantity || gotCounted != counted || gotCapped != capped {
		t.Fatalf("entry %d scores = (%v, %v, %v), want (%v, %v, %v)", entryID, gotQuantity, gotCounted, gotCapped, quantity, counted, capped)
	}
}

func assertFinalScore(t *testing.T, db *Store, entryID int, want float64) {
	t.Helper()
	var got float64
	if err := db.QueryRowContext(context.Background(), `SELECT final_score FROM kpi_entries WHERE id = ?`, entryID).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("entry %d final = %v, want %v", entryID, got, want)
	}
}
