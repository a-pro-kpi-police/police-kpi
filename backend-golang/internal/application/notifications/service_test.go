package notifications

import (
	"context"
	"errors"
	"testing"

	domainnotifications "kpi-police/backend-golang/internal/domain/notifications"
)

func TestListNotificationsUsesDefaultLimit(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo, domainnotifications.NewPolicy())

	if _, err := service.ListNotifications(context.Background(), 7, 0); err != nil {
		t.Fatalf("ListNotifications error = %v", err)
	}
	if repo.listLimit != domainnotifications.DefaultLimit {
		t.Fatalf("limit = %d, want %d", repo.listLimit, domainnotifications.DefaultLimit)
	}
	if repo.listUserID != 7 {
		t.Fatalf("user id = %d, want 7", repo.listUserID)
	}
}

func TestListNotificationsRejectsInvalidLimit(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo, domainnotifications.NewPolicy())

	_, err := service.ListNotifications(context.Background(), 7, domainnotifications.MaxLimit+1)
	if !errors.Is(err, domainnotifications.ErrInvalidLimit) {
		t.Fatalf("error = %v, want %v", err, domainnotifications.ErrInvalidLimit)
	}
	if repo.listCalled {
		t.Fatal("repository was called after invalid limit")
	}
}

type fakeRepository struct {
	listCalled bool
	listUserID int
	listLimit  int
}

func (r *fakeRepository) List(_ context.Context, userID int, limit int) (domainnotifications.List, error) {
	r.listCalled = true
	r.listUserID = userID
	r.listLimit = limit
	return domainnotifications.List{}, nil
}

func (r *fakeRepository) MarkRead(_ context.Context, _ int, _ int) (domainnotifications.Notification, error) {
	return domainnotifications.Notification{}, nil
}

func (r *fakeRepository) MarkAllRead(_ context.Context, _ int) (int64, error) {
	return 0, nil
}
