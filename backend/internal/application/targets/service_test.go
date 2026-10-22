package targets

import (
	"context"
	"errors"
	"reflect"
	"testing"

	domaintargets "kpi-police/backend/internal/domain/targets"
)

func TestSaveOfficerTargetsValidatesAndSaves(t *testing.T) {
	repo := &fakeRepository{
		officer:         Officer{ID: 7},
		validCatalogIDs: map[int]bool{1: true, 2: true},
	}
	service := NewService(repo, domaintargets.NewPolicy())

	err := service.SaveOfficerTargets(context.Background(), 7, "2026-07", []domaintargets.SaveItem{
		{CatalogID: 1, TargetQuantity: 10},
		{CatalogID: 2, TargetQuantity: 0},
	})
	if err != nil {
		t.Fatal(err)
	}
	if repo.savedOfficerID != 7 || repo.savedMonth != "2026-07" {
		t.Fatalf("saved target = (%v, %q), want (7, 2026-07)", repo.savedOfficerID, repo.savedMonth)
	}
	want := map[int]float64{1: 10, 2: 0}
	if !reflect.DeepEqual(repo.savedQuantities, want) {
		t.Fatalf("quantities = %#v, want %#v", repo.savedQuantities, want)
	}
}

func TestSaveOfficerTargetsRejectsUnknownCatalog(t *testing.T) {
	repo := &fakeRepository{
		officer:         Officer{ID: 7},
		validCatalogIDs: map[int]bool{1: true},
	}
	service := NewService(repo, domaintargets.NewPolicy())

	err := service.SaveOfficerTargets(context.Background(), 7, "2026-07", []domaintargets.SaveItem{
		{CatalogID: 2, TargetQuantity: 10},
	})
	if !errors.Is(err, domaintargets.ErrCatalogNotFound) {
		t.Fatalf("error = %v, want %v", err, domaintargets.ErrCatalogNotFound)
	}
	if repo.savedQuantities != nil {
		t.Fatal("targets were saved after validation error")
	}
}

func TestSaveGroupTargetsRejectsEmptyGroup(t *testing.T) {
	repo := &fakeRepository{
		validCatalogIDs: map[int]bool{1: true},
	}
	service := NewService(repo, domaintargets.NewPolicy())

	err := service.SaveGroupTargets(context.Background(), 3, "2026-07", []domaintargets.SaveItem{
		{CatalogID: 1, TargetQuantity: 10},
	})
	if !errors.Is(err, domaintargets.ErrEmptyGroup) {
		t.Fatalf("error = %v, want %v", err, domaintargets.ErrEmptyGroup)
	}
}

func TestSaveGroupTargetsSavesAllOfficers(t *testing.T) {
	repo := &fakeRepository{
		districtOfficerIDs: []int{7, 8},
		validCatalogIDs:    map[int]bool{1: true},
	}
	service := NewService(repo, domaintargets.NewPolicy())

	err := service.SaveGroupTargets(context.Background(), 3, "2026-07", []domaintargets.SaveItem{
		{CatalogID: 1, TargetQuantity: 10},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(repo.savedOfficerIDs, []int{7, 8}) {
		t.Fatalf("officer ids = %#v, want [7 8]", repo.savedOfficerIDs)
	}
}

type fakeRepository struct {
	officer            Officer
	districtOfficerIDs []int
	validCatalogIDs    map[int]bool
	savedOfficerID     int
	savedOfficerIDs    []int
	savedMonth         string
	savedQuantities    map[int]float64
	officerErr         error
	districtErr        error
	catalogErr         error
	saveErr            error
}

func (r *fakeRepository) GetOfficer(context.Context, int) (Officer, error) {
	if r.officerErr != nil {
		return Officer{}, r.officerErr
	}
	return r.officer, nil
}

func (r *fakeRepository) DistrictOfficerIDs(context.Context, int) ([]int, error) {
	if r.districtErr != nil {
		return nil, r.districtErr
	}
	ids := make([]int, len(r.districtOfficerIDs))
	copy(ids, r.districtOfficerIDs)
	return ids, nil
}

func (r *fakeRepository) ActivePositiveCatalogIDs(context.Context) (map[int]bool, error) {
	if r.catalogErr != nil {
		return nil, r.catalogErr
	}
	ids := make(map[int]bool, len(r.validCatalogIDs))
	for id, valid := range r.validCatalogIDs {
		ids[id] = valid
	}
	return ids, nil
}

func (r *fakeRepository) SaveOfficerTargets(_ context.Context, officerID int, month string, quantities map[int]float64) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.savedOfficerID = officerID
	r.savedMonth = month
	r.savedQuantities = copyQuantities(quantities)
	return nil
}

func (r *fakeRepository) SaveGroupTargets(_ context.Context, officerIDs []int, month string, quantities map[int]float64) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.savedOfficerIDs = append([]int(nil), officerIDs...)
	r.savedMonth = month
	r.savedQuantities = copyQuantities(quantities)
	return nil
}

func copyQuantities(values map[int]float64) map[int]float64 {
	result := make(map[int]float64, len(values))
	for catalogID, quantity := range values {
		result[catalogID] = quantity
	}
	return result
}
