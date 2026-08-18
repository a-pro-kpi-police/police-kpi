package targets

import (
	"context"

	domaintargets "kpi-police/backend-golang/internal/domain/targets"
)

type Officer struct {
	ID int
}

type Repository interface {
	GetOfficer(ctx context.Context, id int) (Officer, error)
	DistrictOfficerIDs(ctx context.Context, districtID int) ([]int, error)
	ActivePositiveCatalogIDs(ctx context.Context) (map[int]bool, error)
	SaveOfficerTargets(ctx context.Context, officerID int, month string, quantities map[int]float64) error
	SaveGroupTargets(ctx context.Context, officerIDs []int, month string, quantities map[int]float64) error
}

type Service struct {
	repo   Repository
	policy domaintargets.Policy
}

func NewService(repo Repository, policy domaintargets.Policy) Service {
	return Service{repo: repo, policy: policy}
}

func (s Service) SaveOfficerTargets(ctx context.Context, officerID int, month string, items []domaintargets.SaveItem) error {
	if err := s.policy.ValidateMonth(month); err != nil {
		return err
	}
	officer, err := s.repo.GetOfficer(ctx, officerID)
	if err != nil {
		return err
	}
	quantities, err := s.quantities(ctx, items)
	if err != nil {
		return err
	}
	return s.repo.SaveOfficerTargets(ctx, officer.ID, month, quantities)
}

func (s Service) SaveGroupTargets(ctx context.Context, districtID int, month string, items []domaintargets.SaveItem) error {
	if err := s.policy.ValidateMonth(month); err != nil {
		return err
	}
	officerIDs, err := s.repo.DistrictOfficerIDs(ctx, districtID)
	if err != nil {
		return err
	}
	if len(officerIDs) == 0 {
		return domaintargets.ErrEmptyGroup
	}
	quantities, err := s.quantities(ctx, items)
	if err != nil {
		return err
	}
	return s.repo.SaveGroupTargets(ctx, officerIDs, month, quantities)
}

func (s Service) quantities(ctx context.Context, items []domaintargets.SaveItem) (map[int]float64, error) {
	validCatalogIDs, err := s.repo.ActivePositiveCatalogIDs(ctx)
	if err != nil {
		return nil, err
	}
	return s.policy.TargetQuantities(items, validCatalogIDs)
}
