package messenger

import (
	"errors"
	"testing"
	"time"
)

func TestConversationAllowsHeadAndOfficer(t *testing.T) {
	conversation, err := NewPolicy().Conversation(
		User{ID: 1, Role: RoleHead},
		User{ID: 2, Role: RoleOfficer},
	)
	if err != nil {
		t.Fatalf("Conversation error = %v", err)
	}
	if conversation.OfficerID != 2 {
		t.Fatalf("officer id = %d, want 2", conversation.OfficerID)
	}
}

func TestConversationRejectsSameRole(t *testing.T) {
	_, err := NewPolicy().Conversation(
		User{ID: 1, Role: RoleOfficer},
		User{ID: 2, Role: RoleOfficer},
	)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("error = %v, want %v", err, ErrForbidden)
	}
}

func TestConversationRejectsDifferentDistricts(t *testing.T) {
	leftDistrict := 1
	rightDistrict := 2
	_, err := NewPolicy().Conversation(
		User{ID: 1, Role: RoleHead, DistrictID: &leftDistrict},
		User{ID: 2, Role: RoleOfficer, DistrictID: &rightDistrict},
	)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("error = %v, want %v", err, ErrForbidden)
	}
}

func TestBuildMessageTrimsBodyAndRejectsEmpty(t *testing.T) {
	policy := NewPolicy()
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	message, _, err := policy.BuildMessage(
		User{ID: 1, Role: RoleHead},
		User{ID: 2, Role: RoleOfficer},
		SendRequest{RecipientID: 2, Body: "  hello  "},
		now,
	)
	if err != nil {
		t.Fatalf("BuildMessage error = %v", err)
	}
	if message.Body != "hello" {
		t.Fatalf("body = %q, want hello", message.Body)
	}

	_, _, err = policy.BuildMessage(
		User{ID: 1, Role: RoleHead},
		User{ID: 2, Role: RoleOfficer},
		SendRequest{RecipientID: 2, Body: "   "},
		now,
	)
	if !errors.Is(err, ErrEmptyBody) {
		t.Fatalf("error = %v, want %v", err, ErrEmptyBody)
	}
}

func TestValidateDeleteAllowsOnlySender(t *testing.T) {
	policy := NewPolicy()
	err := policy.ValidateDelete(User{ID: 1}, Message{ID: 10, SenderID: 1})
	if err != nil {
		t.Fatalf("ValidateDelete error = %v", err)
	}

	err = policy.ValidateDelete(User{ID: 2}, Message{ID: 10, SenderID: 1})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("error = %v, want %v", err, ErrForbidden)
	}
}
