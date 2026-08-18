package notifications

import (
	"context"

	domainnotifications "kpi-police/backend-golang/internal/domain/notifications"
)

type Repository interface {
	List(ctx context.Context, userID int, limit int) (domainnotifications.List, error)
	MarkRead(ctx context.Context, userID int, notificationID int) (domainnotifications.Notification, error)
	MarkAllRead(ctx context.Context, userID int) (int64, error)
}

type Service struct {
	repo   Repository
	policy domainnotifications.Policy
}

func NewService(repo Repository, policy domainnotifications.Policy) Service {
	return Service{repo: repo, policy: policy}
}

func (s Service) ListNotifications(ctx context.Context, userID int, limit int) (domainnotifications.List, error) {
	resolvedLimit, err := s.policy.ResolveLimit(limit)
	if err != nil {
		return domainnotifications.List{}, err
	}
	return s.repo.List(ctx, userID, resolvedLimit)
}

func (s Service) MarkRead(ctx context.Context, userID int, notificationID int) (domainnotifications.Notification, error) {
	return s.repo.MarkRead(ctx, userID, notificationID)
}

func (s Service) MarkAllRead(ctx context.Context, userID int) (int64, error) {
	return s.repo.MarkAllRead(ctx, userID)
}
