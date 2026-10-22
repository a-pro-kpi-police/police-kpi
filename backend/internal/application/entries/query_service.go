package entries

import (
	"context"

	domainentries "kpi-police/backend/internal/domain/entries"
)

type QueryRepository interface {
	ListEntries(ctx context.Context, query domainentries.EntryQuery) ([]domainentries.ListedEntry, error)
	GetEntry(ctx context.Context, id int) (domainentries.ListedEntry, error)
}

type QueryService struct {
	repo   QueryRepository
	policy domainentries.QueryPolicy
}

func NewQueryService(repo QueryRepository, policy domainentries.QueryPolicy) QueryService {
	return QueryService{repo: repo, policy: policy}
}

func (s QueryService) ListEntries(ctx context.Context, query domainentries.EntryQuery) ([]domainentries.ListedEntry, error) {
	normalized, err := s.policy.Normalize(query)
	if err != nil {
		return nil, err
	}
	return s.repo.ListEntries(ctx, normalized)
}

func (s QueryService) GetEntry(ctx context.Context, id int) (domainentries.ListedEntry, error) {
	return s.repo.GetEntry(ctx, id)
}
