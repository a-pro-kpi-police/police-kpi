package dashboard

import (
	"context"

	domaindashboard "kpi-police/backend-golang/internal/domain/dashboard"
	domainusers "kpi-police/backend-golang/internal/domain/users"
)

type Repository interface {
	ListEntries(ctx context.Context, filter domaindashboard.Filter) ([]domaindashboard.Entry, error)
	ListCatalogBlocks(ctx context.Context) ([]string, error)
}

type UserRepository interface {
	ListOfficers(ctx context.Context, districtID int) ([]domainusers.User, error)
}

type Service struct {
	repo   Repository
	users  UserRepository
	policy domaindashboard.Policy
}

func NewService(repo Repository, users UserRepository, policy domaindashboard.Policy) Service {
	return Service{repo: repo, users: users, policy: policy}
}

func (s Service) BuildDashboard(ctx context.Context, filter domaindashboard.Filter) (domaindashboard.Report, error) {
	entries, err := s.repo.ListEntries(ctx, filter)
	if err != nil {
		return domaindashboard.Report{}, err
	}
	officers, err := s.users.ListOfficers(ctx, optionalIDValue(filter.DistrictID))
	if err != nil {
		return domaindashboard.Report{}, err
	}
	blocks, err := s.repo.ListCatalogBlocks(ctx)
	if err != nil {
		return domaindashboard.Report{}, err
	}
	reportOfficers := dashboardOfficers(officers)
	report := s.policy.BuildReport(entries, reportOfficers, blocks, filter)

	comparisonEntries := entries
	if filter.Month != "" || filter.OfficerID != nil {
		comparisonFilter := domaindashboard.Filter{DistrictID: filter.DistrictID}
		comparisonEntries, err = s.repo.ListEntries(ctx, comparisonFilter)
		if err != nil {
			return domaindashboard.Report{}, err
		}
	}
	report.Comparisons = s.policy.BuildComparison(
		comparisonEntries,
		filter.Month,
		filter.OfficerID,
		reportOfficers,
	)
	return report, nil
}

func dashboardOfficers(users []domainusers.User) []domaindashboard.Officer {
	result := make([]domaindashboard.Officer, 0, len(users))
	for _, user := range users {
		officer := domaindashboard.Officer{
			ID:       user.ID,
			FullName: user.FullName,
			Rank:     user.Rank,
		}
		if user.District != nil {
			officer.DistrictName = user.District.Name
		}
		result = append(result, officer)
	}
	return result
}

func optionalIDValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
