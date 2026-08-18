package targets

import (
	"context"

	domaintargets "kpi-police/backend-golang/internal/domain/targets"
)

type TargetQueryRepository interface {
	GetOfficerProfile(ctx context.Context, id int) (domaintargets.Officer, error)
	GetDistrict(ctx context.Context, id int) (domaintargets.District, error)
	ListOfficerProfiles(ctx context.Context, districtID int) ([]domaintargets.Officer, error)
	ListActivePositiveCatalog(ctx context.Context) ([]domaintargets.CatalogItem, error)
	OfficerTargetQuantities(ctx context.Context, officerID int, month string) (map[int]float64, error)
	GroupTargetQuantities(ctx context.Context, officerIDs []int, month string) (map[int]map[int]float64, error)
	ActualsByCatalog(ctx context.Context, officerID int, month string) (map[int]domaintargets.Actual, error)
	PenaltiesByOfficer(ctx context.Context, month string, officerIDs []int) (map[int]domaintargets.Penalty, error)
}

type QueryService struct {
	repo   TargetQueryRepository
	policy domaintargets.Policy
}

func NewQueryService(repo TargetQueryRepository, policy domaintargets.Policy) QueryService {
	return QueryService{repo: repo, policy: policy}
}

func (s QueryService) TargetRows(ctx context.Context, officerID int, month string) ([]domaintargets.Target, error) {
	if err := s.policy.ValidateMonth(month); err != nil {
		return nil, err
	}
	officer, err := s.repo.GetOfficerProfile(ctx, officerID)
	if err != nil {
		return nil, err
	}
	return s.targetRowsForOfficer(ctx, officer, month)
}

func (s QueryService) GroupTargetRows(ctx context.Context, districtID int, month string) ([]domaintargets.GroupTarget, error) {
	if err := s.policy.ValidateMonth(month); err != nil {
		return nil, err
	}
	district, err := s.repo.GetDistrict(ctx, districtID)
	if err != nil {
		return nil, err
	}
	officers, err := s.repo.ListOfficerProfiles(ctx, district.ID)
	if err != nil {
		return nil, err
	}
	officerIDs := officerIDs(officers)
	quantities, err := s.repo.GroupTargetQuantities(ctx, officerIDs, month)
	if err != nil {
		return nil, err
	}
	catalog, err := s.repo.ListActivePositiveCatalog(ctx)
	if err != nil {
		return nil, err
	}
	return s.policy.BuildGroupTargets(catalog, quantities, officerIDs, district.Coeff), nil
}

func (s QueryService) Progress(ctx context.Context, officerID int, month string) (domaintargets.Progress, error) {
	if err := s.policy.ValidateMonth(month); err != nil {
		return domaintargets.Progress{}, err
	}
	officer, err := s.repo.GetOfficerProfile(ctx, officerID)
	if err != nil {
		return domaintargets.Progress{}, err
	}
	return s.progressForOfficer(ctx, officer, month)
}

func (s QueryService) Overview(ctx context.Context, filter domaintargets.OverviewFilter) (domaintargets.Overview, error) {
	if err := s.policy.ValidateMonth(filter.Month); err != nil {
		return domaintargets.Overview{}, err
	}
	if filter.DistrictID != nil {
		if _, err := s.repo.GetDistrict(ctx, *filter.DistrictID); err != nil {
			return domaintargets.Overview{}, err
		}
	}

	var officers []domaintargets.Officer
	var err error
	if filter.OfficerID != nil {
		officer, err := s.repo.GetOfficerProfile(ctx, *filter.OfficerID)
		if err != nil {
			return domaintargets.Overview{}, err
		}
		officers = []domaintargets.Officer{officer}
	} else {
		districtID := 0
		if filter.DistrictID != nil {
			districtID = *filter.DistrictID
		}
		officers, err = s.repo.ListOfficerProfiles(ctx, districtID)
		if err != nil {
			return domaintargets.Overview{}, err
		}
	}

	penalties, err := s.repo.PenaltiesByOfficer(ctx, filter.Month, officerIDs(officers))
	if err != nil {
		return domaintargets.Overview{}, err
	}
	inputs := make([]domaintargets.OverviewInput, 0, len(officers))
	for _, officer := range officers {
		progress, err := s.progressForOfficer(ctx, officer, filter.Month)
		if err != nil {
			return domaintargets.Overview{}, err
		}
		inputs = append(inputs, domaintargets.OverviewInput{
			OfficerID: officer.ID,
			Name:      officer.FullName,
			Progress:  progress,
			Penalty:   penalties[officer.ID],
		})
	}
	return s.policy.BuildOverview(filter.Month, inputs, filter.OfficerID != nil), nil
}

func (s QueryService) targetRowsForOfficer(ctx context.Context, officer domaintargets.Officer, month string) ([]domaintargets.Target, error) {
	coeff := 1.0
	if officer.District != nil {
		coeff = officer.District.Coeff
	}
	quantities, err := s.repo.OfficerTargetQuantities(ctx, officer.ID, month)
	if err != nil {
		return nil, err
	}
	catalog, err := s.repo.ListActivePositiveCatalog(ctx)
	if err != nil {
		return nil, err
	}
	return s.policy.BuildTargets(catalog, quantities, coeff), nil
}

func (s QueryService) progressForOfficer(ctx context.Context, officer domaintargets.Officer, month string) (domaintargets.Progress, error) {
	targets, err := s.targetRowsForOfficer(ctx, officer, month)
	if err != nil {
		return domaintargets.Progress{}, err
	}
	actuals, err := s.repo.ActualsByCatalog(ctx, officer.ID, month)
	if err != nil {
		return domaintargets.Progress{}, err
	}
	return s.policy.BuildProgress(month, officer.ID, officer.MonthlyNorm, targets, actuals), nil
}

func officerIDs(officers []domaintargets.Officer) []int {
	result := make([]int, 0, len(officers))
	for _, officer := range officers {
		result = append(result, officer.ID)
	}
	return result
}
