package users

import (
	"context"

	domainusers "kpi-police/backend-golang/internal/domain/users"
)

type QueryRepository interface {
	ListDistricts(ctx context.Context) ([]domainusers.District, error)
	ListOfficers(ctx context.Context, districtID int) ([]domainusers.User, error)
	ConfirmedScores(ctx context.Context) (map[int]float64, error)
	ListMonths(ctx context.Context, officerID *int, districtID *int) ([]string, error)
}

type QueryService struct {
	repo   QueryRepository
	policy domainusers.EfficiencyPolicy
}

func NewQueryService(repo QueryRepository, policy domainusers.EfficiencyPolicy) QueryService {
	return QueryService{repo: repo, policy: policy}
}

func (s QueryService) ListDistricts(ctx context.Context) ([]domainusers.District, error) {
	return s.repo.ListDistricts(ctx)
}

func (s QueryService) ListOfficersWithEfficiency(ctx context.Context, districtID int) ([]domainusers.OfficerWithEfficiency, error) {
	officers, err := s.repo.ListOfficers(ctx, districtID)
	if err != nil {
		return nil, err
	}
	scores, err := s.repo.ConfirmedScores(ctx)
	if err != nil {
		return nil, err
	}
	result, _ := s.policy.RankOfficers(officers, scores)
	return result, nil
}

func (s QueryService) EfficiencyMap(ctx context.Context, districtID int) (map[int]domainusers.Efficiency, error) {
	officers, err := s.repo.ListOfficers(ctx, districtID)
	if err != nil {
		return nil, err
	}
	scores, err := s.repo.ConfirmedScores(ctx)
	if err != nil {
		return nil, err
	}
	_, result := s.policy.RankOfficers(officers, scores)
	return result, nil
}

func (s QueryService) ListMonths(ctx context.Context, actor domainusers.User) ([]string, error) {
	var officerID *int
	var districtID *int
	if actor.Role == domainusers.RoleOfficer {
		id := actor.ID
		officerID = &id
	} else if (actor.Role == domainusers.RoleHead || actor.Role == domainusers.RoleLeadership) && actor.DistrictID != nil {
		districtID = actor.DistrictID
	}
	return s.repo.ListMonths(ctx, officerID, districtID)
}
