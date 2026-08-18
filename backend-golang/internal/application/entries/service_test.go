package entries

import (
	"context"
	"errors"
	"testing"

	domainentries "kpi-police/backend-golang/internal/domain/entries"
)

func TestCreateEntryBuildsAndSavesPendingEntry(t *testing.T) {
	quantity := 8.0
	repo := &fakeRepository{
		officer: domainentries.Officer{ID: 7, Coeff: 1.2},
		catalog: domainentries.Catalog{ID: 3, Code: "A590", PointsPerUnit: 5, MonthlyLimit: 6, IsActive: true},
	}
	service := NewService(repo, domainentries.NewCreationPolicy(), func() string { return "2026-07-15" })

	entry, err := service.CreateEntry(context.Background(),
		domainentries.Actor{ID: 7, FullName: "Officer", Role: domainentries.RoleOfficer},
		domainentries.CreateRequest{CatalogID: 3, Quantity: &quantity},
	)
	if err != nil {
		t.Fatal(err)
	}
	if entry.ID != 99 {
		t.Fatalf("entry id = %v, want 99", entry.ID)
	}
	if repo.saved.OfficerID != 7 || repo.saved.CatalogID != 3 {
		t.Fatalf("saved identity = (%v, %v), want (7, 3)", repo.saved.OfficerID, repo.saved.CatalogID)
	}
	if repo.saved.CountedQuantity != 6 || repo.saved.FinalScore != 36 {
		t.Fatalf("saved scores = (%v, %v), want (6, 36)", repo.saved.CountedQuantity, repo.saved.FinalScore)
	}
}

func TestCreateEntryReturnsCatalogErrorBeforeSaving(t *testing.T) {
	catalogErr := errors.New("catalog failed")
	repo := &fakeRepository{
		officer:    domainentries.Officer{ID: 7, Coeff: 1},
		catalogErr: catalogErr,
	}
	service := NewService(repo, domainentries.NewCreationPolicy(), func() string { return "2026-07-15" })

	_, err := service.CreateEntry(context.Background(),
		domainentries.Actor{ID: 7, Role: domainentries.RoleOfficer},
		domainentries.CreateRequest{CatalogID: 3},
	)
	if !errors.Is(err, catalogErr) {
		t.Fatalf("error = %v, want %v", err, catalogErr)
	}
	if repo.saved.ID != 0 {
		t.Fatal("entry was saved after catalog error")
	}
}

func TestUpdateEntryStatusAppliesChange(t *testing.T) {
	repo := &fakeStatusRepository{
		entry: domainentries.Entry{
			ID:        10,
			OfficerID: 7,
			Month:     "2026-07",
			Status:    domainentries.StatusPending,
		},
	}
	service := NewStatusService(repo, domainentries.NewStatusPolicy())

	entry, err := service.UpdateEntryStatus(context.Background(),
		domainentries.Actor{ID: 1, Role: domainentries.RoleHead},
		domainentries.StatusUpdateRequest{EntryID: 10, Status: domainentries.StatusConfirmed, HeadComment: "ok"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if entry.Status != domainentries.StatusConfirmed {
		t.Fatalf("status = %q, want confirmed", entry.Status)
	}
	if !repo.applied.NotifyOfficer {
		t.Fatal("notify officer = false, want true")
	}
	if repo.applied.Entry.HeadComment != "ok" {
		t.Fatalf("head comment = %q, want ok", repo.applied.Entry.HeadComment)
	}
}

func TestUpdateEntryStatusReturnsLoadError(t *testing.T) {
	loadErr := errors.New("load failed")
	repo := &fakeStatusRepository{loadErr: loadErr}
	service := NewStatusService(repo, domainentries.NewStatusPolicy())

	_, err := service.UpdateEntryStatus(context.Background(),
		domainentries.Actor{ID: 1, Role: domainentries.RoleHead},
		domainentries.StatusUpdateRequest{EntryID: 10, Status: domainentries.StatusConfirmed},
	)
	if !errors.Is(err, loadErr) {
		t.Fatalf("error = %v, want %v", err, loadErr)
	}
	if repo.applied.Entry.ID != 0 {
		t.Fatal("status change was applied after load error")
	}
}

type fakeRepository struct {
	officer    domainentries.Officer
	catalog    domainentries.Catalog
	saved      domainentries.Entry
	officerErr error
	catalogErr error
	saveErr    error
}

func (r *fakeRepository) GetOfficer(context.Context, int) (domainentries.Officer, error) {
	if r.officerErr != nil {
		return domainentries.Officer{}, r.officerErr
	}
	return r.officer, nil
}

func (r *fakeRepository) GetCatalog(context.Context, int) (domainentries.Catalog, error) {
	if r.catalogErr != nil {
		return domainentries.Catalog{}, r.catalogErr
	}
	return r.catalog, nil
}

func (r *fakeRepository) CreateEntry(_ context.Context, _ domainentries.Actor, entry domainentries.Entry) (domainentries.Entry, error) {
	if r.saveErr != nil {
		return domainentries.Entry{}, r.saveErr
	}
	r.saved = entry
	entry.ID = 99
	return entry, nil
}

type fakeStatusRepository struct {
	entry    domainentries.Entry
	applied  domainentries.StatusChange
	loadErr  error
	applyErr error
}

func (r *fakeStatusRepository) GetEntryForStatusUpdate(context.Context, int) (domainentries.Entry, error) {
	if r.loadErr != nil {
		return domainentries.Entry{}, r.loadErr
	}
	return r.entry, nil
}

func (r *fakeStatusRepository) ApplyStatusChange(_ context.Context, change domainentries.StatusChange) (domainentries.Entry, error) {
	if r.applyErr != nil {
		return domainentries.Entry{}, r.applyErr
	}
	r.applied = change
	return change.Entry, nil
}
