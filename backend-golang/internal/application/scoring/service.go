package scoring

import (
	"context"

	domainscoring "kpi-police/backend-golang/internal/domain/scoring"
)

type EntryRepository interface {
	ListOfficerMonth(ctx context.Context, officerID int, month string) ([]domainscoring.Entry, error)
	SaveScoredEntries(ctx context.Context, entries []domainscoring.Entry) error
}

type MonthResult struct {
	OfficerID int
	Month     string
	Result    domainscoring.Result
}

type Service struct {
	repo   EntryRepository
	policy domainscoring.Policy
}

func NewService(repo EntryRepository, policy domainscoring.Policy) Service {
	return Service{repo: repo, policy: policy}
}

func (s Service) RecalculateOfficerMonth(ctx context.Context, officerID int, month string) (MonthResult, error) {
	entries, err := s.repo.ListOfficerMonth(ctx, officerID, month)
	if err != nil {
		return MonthResult{}, err
	}
	calculation := s.policy.Calculate(entries)
	if err := s.repo.SaveScoredEntries(ctx, calculation.Entries); err != nil {
		return MonthResult{}, err
	}
	return MonthResult{
		OfficerID: officerID,
		Month:     month,
		Result:    calculation.Result,
	}, nil
}
