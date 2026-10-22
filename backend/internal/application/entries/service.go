package entries

import (
	"context"

	domainentries "kpi-police/backend/internal/domain/entries"
)

type Repository interface {
	GetOfficer(ctx context.Context, id int) (domainentries.Officer, error)
	GetCatalog(ctx context.Context, id int) (domainentries.Catalog, error)
	CreateEntry(ctx context.Context, actor domainentries.Actor, entry domainentries.Entry) (domainentries.Entry, error)
}

type StatusRepository interface {
	GetEntryForStatusUpdate(ctx context.Context, id int) (domainentries.Entry, error)
	ApplyStatusChange(ctx context.Context, change domainentries.StatusChange) (domainentries.Entry, error)
}

type Service struct {
	repo   Repository
	policy domainentries.CreationPolicy
	today  func() string
}

func NewService(repo Repository, policy domainentries.CreationPolicy, today func() string) Service {
	return Service{repo: repo, policy: policy, today: today}
}

func (s Service) CreateEntry(ctx context.Context, actor domainentries.Actor, request domainentries.CreateRequest) (domainentries.Entry, error) {
	officerID, err := s.policy.ResolveOfficerID(actor, request.OfficerID)
	if err != nil {
		return domainentries.Entry{}, err
	}
	officer, err := s.repo.GetOfficer(ctx, officerID)
	if err != nil {
		return domainentries.Entry{}, err
	}
	catalog, err := s.repo.GetCatalog(ctx, request.CatalogID)
	if err != nil {
		return domainentries.Entry{}, err
	}
	entry, err := s.policy.BuildPendingEntry(request, actor, officer, catalog, s.today())
	if err != nil {
		return domainentries.Entry{}, err
	}
	return s.repo.CreateEntry(ctx, actor, entry)
}

type StatusService struct {
	repo   StatusRepository
	policy domainentries.StatusPolicy
}

func NewStatusService(repo StatusRepository, policy domainentries.StatusPolicy) StatusService {
	return StatusService{repo: repo, policy: policy}
}

func (s StatusService) UpdateEntryStatus(ctx context.Context, actor domainentries.Actor, request domainentries.StatusUpdateRequest) (domainentries.Entry, error) {
	entry, err := s.repo.GetEntryForStatusUpdate(ctx, request.EntryID)
	if err != nil {
		return domainentries.Entry{}, err
	}
	change, err := s.policy.ChangeStatus(actor, entry, request)
	if err != nil {
		return domainentries.Entry{}, err
	}
	return s.repo.ApplyStatusChange(ctx, change)
}
