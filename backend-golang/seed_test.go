package main

import (
	"context"
	"path/filepath"
	"testing"
)

func TestMonthlyImportSeed(t *testing.T) {
	if testing.Short() {
		t.Skip("monthly CSV import is an integration check")
	}
	dbPath := filepath.ToSlash(filepath.Join(t.TempDir(), "kpi_july_2026.db"))
	t.Setenv("KPI_DATABASE_URL", "sqlite:///"+dbPath)
	t.Setenv("KPI_SEED_MODE", monthlyImportMode)
	t.Setenv("KPI_UPLOAD_DIR", filepath.Join(t.TempDir(), "uploads"))

	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()

	assertCount(t, app, "users", 153)
	assertCount(t, app, "kpi_entries", 20368)

	var activeCatalog int
	if err := app.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM kpi_catalog WHERE is_active = 1`).Scan(&activeCatalog); err != nil {
		t.Fatal(err)
	}
	if activeCatalog != 134 {
		t.Fatalf("active catalog count = %d, want 134", activeCatalog)
	}
	head, err := app.getUserByUsername(context.Background(), "head")
	if err != nil {
		t.Fatal(err)
	}
	if !verifyPassword("head123", head.PasswordHash) {
		t.Fatal("seeded head password is not valid")
	}
}

func assertCount(t *testing.T, app *App, table string, want int) {
	t.Helper()
	var got int
	if err := app.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM `+table).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s count = %d, want %d", table, got, want)
	}
}
