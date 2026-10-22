package scoring

import (
	"context"
	"errors"
	"testing"

	domainscoring "kpi-police/backend/internal/domain/scoring"
)

func TestRecalculateOfficerMonthSavesCalculatedEntries(t *testing.T) {
	repo := &fakeEntryRepository{
		entries: []domainscoring.Entry{
			confirmedEntry(1, 1, 35, 1, 20, true, false),
			confirmedEntry(2, 2, 8, 5, 6, false, false),
		},
	}
	service := NewService(repo, domainscoring.NewPolicy())

	result, err := service.RecalculateOfficerMonth(context.Background(), 7, "2026-07")
	if err != nil {
		t.Fatal(err)
	}

	if result.OfficerID != 7 || result.Month != "2026-07" {
		t.Fatalf("result identity = (%v, %q), want (7, 2026-07)", result.OfficerID, result.Month)
	}
	if result.Result.FinalScore != 50 {
		t.Fatalf("final score = %v, want 50", result.Result.FinalScore)
	}
	if len(repo.saved) != 2 {
		t.Fatalf("saved entries = %d, want 2", len(repo.saved))
	}
	if repo.saved[0].CountedQuantity != 20 || repo.saved[1].CappedScore != 30 {
		t.Fatalf("saved scores = (%v, %v), want (20, 30)", repo.saved[0].CountedQuantity, repo.saved[1].CappedScore)
	}
}

func TestRecalculateOfficerMonthReturnsLoadError(t *testing.T) {
	loadErr := errors.New("load failed")
	repo := &fakeEntryRepository{loadErr: loadErr}
	service := NewService(repo, domainscoring.NewPolicy())

	_, err := service.RecalculateOfficerMonth(context.Background(), 7, "2026-07")
	if !errors.Is(err, loadErr) {
		t.Fatalf("error = %v, want %v", err, loadErr)
	}
	if repo.saved != nil {
		t.Fatal("entries were saved after load error")
	}
}

type fakeEntryRepository struct {
	entries []domainscoring.Entry
	saved   []domainscoring.Entry
	loadErr error
	saveErr error
}

func (r *fakeEntryRepository) ListOfficerMonth(context.Context, int, string) ([]domainscoring.Entry, error) {
	if r.loadErr != nil {
		return nil, r.loadErr
	}
	entries := make([]domainscoring.Entry, len(r.entries))
	copy(entries, r.entries)
	return entries, nil
}

func (r *fakeEntryRepository) SaveScoredEntries(_ context.Context, entries []domainscoring.Entry) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.saved = make([]domainscoring.Entry, len(entries))
	copy(r.saved, entries)
	return nil
}

func confirmedEntry(id, catalogID int, quantity, points, limit float64, isMass, isCritical bool) domainscoring.Entry {
	return domainscoring.Entry{
		ID:            id,
		CatalogID:     catalogID,
		Quantity:      quantity,
		PointsPerUnit: points,
		Coeff:         1,
		Status:        domainscoring.StatusConfirmed,
		Catalog: &domainscoring.CatalogSnapshot{
			MonthlyLimit: limit,
			IsMass:       isMass,
			IsCritical:   isCritical,
		},
	}
}
