package entries

import (
	"errors"
	"testing"
)

func TestResolveOfficerIDRequiresOfficerForHead(t *testing.T) {
	_, err := NewCreationPolicy().ResolveOfficerID(Actor{ID: 1, Role: RoleHead}, nil)
	if !errors.Is(err, ErrOfficerRequired) {
		t.Fatalf("error = %v, want %v", err, ErrOfficerRequired)
	}
}

func TestResolveOfficerIDForbidsOfficerCreatingForAnotherOfficer(t *testing.T) {
	otherID := 2
	_, err := NewCreationPolicy().ResolveOfficerID(Actor{ID: 1, Role: RoleOfficer}, &otherID)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("error = %v, want %v", err, ErrForbidden)
	}
}

func TestBuildPendingEntryCapsQuantityAndCalculatesScores(t *testing.T) {
	requestedDate := "2026-07-15"
	quantity := 8.0
	entry, err := NewCreationPolicy().BuildPendingEntry(
		CreateRequest{CatalogID: 3, Quantity: &quantity, Date: &requestedDate, Comments: "done"},
		Actor{ID: 1, Role: RoleOfficer},
		Officer{ID: 1, Coeff: 1.2},
		Catalog{ID: 3, Code: "A590", PointsPerUnit: 5, MonthlyLimit: 6, IsActive: true},
		"2026-07-01",
	)
	if err != nil {
		t.Fatal(err)
	}
	if entry.Month != "2026-07" || entry.Status != StatusPending {
		t.Fatalf("entry month/status = (%q, %q), want (2026-07, pending)", entry.Month, entry.Status)
	}
	if entry.Quantity != 8 || entry.CountedQuantity != 6 || entry.CappedScore != 30 {
		t.Fatalf("entry scores = (%v, %v, %v), want (8, 6, 30)", entry.Quantity, entry.CountedQuantity, entry.CappedScore)
	}
	if entry.FinalScore != 36 {
		t.Fatalf("final score = %v, want 36", entry.FinalScore)
	}
}

func TestBuildPendingEntryForbidsNegativeKPIForOfficer(t *testing.T) {
	_, err := NewCreationPolicy().BuildPendingEntry(
		CreateRequest{CatalogID: 3},
		Actor{ID: 1, Role: RoleOfficer},
		Officer{ID: 1, Coeff: 1},
		Catalog{ID: 3, PointsPerUnit: -5, IsActive: true},
		"2026-07-01",
	)
	if !errors.Is(err, ErrNegativeKPIForbidden) {
		t.Fatalf("error = %v, want %v", err, ErrNegativeKPIForbidden)
	}
}

func TestChangeStatusRejectsInvalidStatus(t *testing.T) {
	_, err := NewStatusPolicy().ChangeStatus(
		Actor{ID: 1, Role: RoleHead},
		Entry{ID: 10, Status: StatusPending},
		StatusUpdateRequest{EntryID: 10, Status: "done"},
	)
	if !errors.Is(err, ErrInvalidStatus) {
		t.Fatalf("error = %v, want %v", err, ErrInvalidStatus)
	}
}

func TestChangeStatusForbidsNonHead(t *testing.T) {
	_, err := NewStatusPolicy().ChangeStatus(
		Actor{ID: 1, Role: RoleOfficer},
		Entry{ID: 10, Status: StatusPending},
		StatusUpdateRequest{EntryID: 10, Status: StatusConfirmed},
	)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("error = %v, want %v", err, ErrForbidden)
	}
}

func TestChangeStatusMarksOfficerNotification(t *testing.T) {
	change, err := NewStatusPolicy().ChangeStatus(
		Actor{ID: 1, Role: RoleHead},
		Entry{ID: 10, Status: StatusPending},
		StatusUpdateRequest{EntryID: 10, Status: StatusConfirmed, HeadComment: "ok"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if change.Entry.Status != StatusConfirmed || change.Entry.HeadComment != "ok" {
		t.Fatalf("updated status/comment = (%q, %q), want (confirmed, ok)", change.Entry.Status, change.Entry.HeadComment)
	}
	if change.PreviousStatus != StatusPending {
		t.Fatalf("previous status = %q, want pending", change.PreviousStatus)
	}
	if !change.NotifyOfficer {
		t.Fatal("notify officer = false, want true")
	}
}
