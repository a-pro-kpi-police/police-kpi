package messenger

import (
	"context"
	"errors"
	"sort"
	"time"

	domainmessenger "kpi-police/backend-golang/internal/domain/messenger"
	domainusers "kpi-police/backend-golang/internal/domain/users"
)

type UserRepository interface {
	GetByID(ctx context.Context, id int) (domainusers.User, error)
	ListByRole(ctx context.Context, role string) ([]domainusers.User, error)
}

type Repository interface {
	ConversationSummaries(ctx context.Context, userID int, contactIDs []int) (map[int]domainmessenger.ConversationSummary, error)
	UnreadCount(ctx context.Context, userID int) (int, error)
	ListEntries(ctx context.Context, officerID int, limit int) ([]domainmessenger.Entry, error)
	EntryOfficerID(ctx context.Context, entryID int) (int, error)
	ConversationMessages(ctx context.Context, userID int, contactID int) ([]domainmessenger.Message, error)
	ConversationMessagesAfter(ctx context.Context, userID int, contactID int, afterID int) ([]domainmessenger.Message, error)
	MarkMessagesRead(ctx context.Context, ids []int, readAt time.Time) error
	CreateMessage(ctx context.Context, message domainmessenger.CreateMessage) (domainmessenger.Message, error)
	GetMessageByID(ctx context.Context, id int) (domainmessenger.Message, error)
	DeleteMessage(ctx context.Context, id int, deletedAt time.Time) (domainmessenger.Message, error)
}

type Service struct {
	users  UserRepository
	repo   Repository
	policy domainmessenger.Policy
	now    func() time.Time
}

func NewService(users UserRepository, repo Repository, policy domainmessenger.Policy, now func() time.Time) Service {
	return Service{users: users, repo: repo, policy: policy, now: now}
}

func (s Service) ListContacts(ctx context.Context, user domainusers.User) ([]domainmessenger.Contact, error) {
	actor := messengerUser(user)
	targetRole := domainusers.RoleHead
	if user.Role == domainusers.RoleHead {
		targetRole = domainusers.RoleOfficer
	}
	users, err := s.users.ListByRole(ctx, targetRole)
	if err != nil {
		return nil, err
	}

	contactIDs := make([]int, 0, len(users))
	contacts := make([]domainmessenger.Contact, 0, len(users))
	for _, candidate := range users {
		contactUser := messengerUser(candidate)
		if _, err := s.policy.Conversation(actor, contactUser); err != nil {
			continue
		}
		contactIDs = append(contactIDs, candidate.ID)
		contacts = append(contacts, domainmessenger.Contact{User: contactUser})
	}
	summaries, err := s.repo.ConversationSummaries(ctx, user.ID, contactIDs)
	if err != nil {
		return nil, err
	}
	for i := range contacts {
		summary := summaries[contacts[i].User.ID]
		contacts[i].UnreadCount = summary.UnreadCount
		contacts[i].LastMessageAt = summary.LastMessageAt
		contacts[i].LastMessageDeleted = summary.LastMessageDeleted
		contacts[i].LastMessage = summary.LastMessage
	}
	sortContacts(contacts)
	return contacts, nil
}

func (s Service) UnreadCount(ctx context.Context, user domainusers.User) (int, error) {
	return s.repo.UnreadCount(ctx, user.ID)
}

func (s Service) ListEntries(ctx context.Context, user domainusers.User, contactID int) ([]domainmessenger.Entry, error) {
	conversation, err := s.conversation(ctx, user, contactID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListEntries(ctx, conversation.OfficerID, 200)
}

func (s Service) ConversationMessages(ctx context.Context, user domainusers.User, contactID int) ([]domainmessenger.Message, error) {
	if _, err := s.conversation(ctx, user, contactID); err != nil {
		return nil, err
	}
	messages, err := s.repo.ConversationMessages(ctx, user.ID, contactID)
	if err != nil {
		return nil, err
	}
	unreadIDs := s.policy.UnreadMessageIDs(messages, user.ID)
	if len(unreadIDs) == 0 {
		return messages, nil
	}
	readAt := s.now().UTC()
	if err := s.repo.MarkMessagesRead(ctx, unreadIDs, readAt); err != nil {
		return nil, err
	}
	return s.policy.ApplyReadAt(messages, user.ID, readAt), nil
}

func (s Service) ConversationMessagesAfter(ctx context.Context, user domainusers.User, contactID int, afterID int) ([]domainmessenger.Message, error) {
	if _, err := s.conversation(ctx, user, contactID); err != nil {
		return nil, err
	}
	messages, err := s.repo.ConversationMessagesAfter(ctx, user.ID, contactID, afterID)
	if err != nil {
		return nil, err
	}
	unreadIDs := s.policy.UnreadMessageIDs(messages, user.ID)
	if len(unreadIDs) == 0 {
		return messages, nil
	}
	readAt := s.now().UTC()
	if err := s.repo.MarkMessagesRead(ctx, unreadIDs, readAt); err != nil {
		return nil, err
	}
	return s.policy.ApplyReadAt(messages, user.ID, readAt), nil
}

func (s Service) SendMessage(ctx context.Context, user domainusers.User, request domainmessenger.SendRequest) (domainmessenger.Message, error) {
	contact, err := s.contact(ctx, request.RecipientID)
	if err != nil {
		return domainmessenger.Message{}, err
	}
	message, conversation, err := s.policy.BuildMessage(messengerUser(user), contact, request, s.now())
	if err != nil {
		return domainmessenger.Message{}, err
	}
	if request.EntryID != nil {
		officerID, err := s.repo.EntryOfficerID(ctx, *request.EntryID)
		if errors.Is(err, domainmessenger.ErrEntryNotFound) {
			return domainmessenger.Message{}, domainmessenger.ErrEntryNotFound
		}
		if err != nil {
			return domainmessenger.Message{}, err
		}
		if err := s.policy.ValidateEntry(conversation, officerID); err != nil {
			return domainmessenger.Message{}, err
		}
	}
	return s.repo.CreateMessage(ctx, message)
}

func (s Service) DeleteMessage(ctx context.Context, user domainusers.User, messageID int) (domainmessenger.Message, error) {
	message, err := s.repo.GetMessageByID(ctx, messageID)
	if err != nil {
		return domainmessenger.Message{}, err
	}
	if err := s.policy.ValidateDelete(messengerUser(user), message); err != nil {
		return domainmessenger.Message{}, err
	}
	if message.DeletedAt != nil {
		return message, nil
	}
	return s.repo.DeleteMessage(ctx, messageID, s.now().UTC())
}

func (s Service) conversation(ctx context.Context, user domainusers.User, contactID int) (domainmessenger.Conversation, error) {
	contact, err := s.contact(ctx, contactID)
	if err != nil {
		return domainmessenger.Conversation{}, err
	}
	return s.policy.Conversation(messengerUser(user), contact)
}

func (s Service) contact(ctx context.Context, id int) (domainmessenger.User, error) {
	contact, err := s.users.GetByID(ctx, id)
	if errors.Is(err, domainusers.ErrNotFound) {
		return domainmessenger.User{}, domainmessenger.ErrContactNotFound
	}
	if err != nil {
		return domainmessenger.User{}, err
	}
	return messengerUser(contact), nil
}

func messengerUser(user domainusers.User) domainmessenger.User {
	result := domainmessenger.User{
		ID:       user.ID,
		FullName: user.FullName,
		Rank:     user.Rank,
		Role:     user.Role,
	}
	if user.DistrictID != nil {
		id := *user.DistrictID
		result.DistrictID = &id
	}
	if user.District != nil {
		result.DistrictName = user.District.Name
	}
	return result
}

func sortContacts(contacts []domainmessenger.Contact) {
	sort.SliceStable(contacts, func(i, j int) bool {
		return contacts[i].User.FullName < contacts[j].User.FullName
	})
	sort.SliceStable(contacts, func(i, j int) bool {
		if contacts[i].LastMessageAt == nil && contacts[j].LastMessageAt == nil {
			return false
		}
		if contacts[i].LastMessageAt == nil {
			return false
		}
		if contacts[j].LastMessageAt == nil {
			return true
		}
		return contacts[i].LastMessageAt.After(*contacts[j].LastMessageAt)
	})
}
