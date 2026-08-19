package messenger

import (
	"errors"
	"strings"
	"time"
)

const (
	RoleHead    = "head"
	RoleOfficer = "officer"

	MaxBodyLength = 2000
)

var (
	ErrContactNotFound = errors.New("contact not found")
	ErrForbidden       = errors.New("conversation forbidden")
	ErrEmptyBody       = errors.New("message body empty")
	ErrBodyTooLong     = errors.New("message body too long")
	ErrEntryNotFound   = errors.New("entry not found")
	ErrMessageNotFound = errors.New("message not found")
)

type User struct {
	ID           int
	FullName     string
	Rank         string
	Role         string
	DistrictName string
}

type Conversation struct {
	User      User
	Contact   User
	OfficerID int
}

type Contact struct {
	User               User
	UnreadCount        int
	LastMessage        string
	LastMessageAt      *time.Time
	LastMessageDeleted bool
}

type ConversationSummary struct {
	ContactID          int
	UnreadCount        int
	LastMessage        string
	LastMessageAt      *time.Time
	LastMessageDeleted bool
}

type Message struct {
	ID          int
	SenderID    int
	RecipientID int
	EntryID     *int
	Body        string
	SenderName  string
	CreatedAt   time.Time
	ReadAt      *time.Time
	DeletedAt   *time.Time
	Entry       *Entry
}

type Entry struct {
	ID              int
	Date            string
	Code            string
	Indicator       string
	Block           string
	NPA             string
	SourceURL       string
	Quantity        float64
	CountedQuantity float64
	MonthlyLimit    float64
	PointsPerUnit   float64
	Coeff           float64
	RawScore        float64
	CappedScore     float64
	AntiCoeff       float64
	FinalScore      float64
	Status          string
	Comments        string
	HeadComment     string
}

type CreateMessage struct {
	SenderID    int
	RecipientID int
	EntryID     *int
	Body        string
	CreatedAt   time.Time
}

type SendRequest struct {
	RecipientID int
	Body        string
	EntryID     *int
}

type Policy struct{}

func NewPolicy() Policy {
	return Policy{}
}

func (Policy) Conversation(user User, contact User) (Conversation, error) {
	switch {
	case user.Role == RoleHead && contact.Role == RoleOfficer:
		return Conversation{User: user, Contact: contact, OfficerID: contact.ID}, nil
	case user.Role == RoleOfficer && contact.Role == RoleHead:
		return Conversation{User: user, Contact: contact, OfficerID: user.ID}, nil
	default:
		return Conversation{}, ErrForbidden
	}
}

func (p Policy) BuildMessage(user User, contact User, request SendRequest, now time.Time) (CreateMessage, Conversation, error) {
	conversation, err := p.Conversation(user, contact)
	if err != nil {
		return CreateMessage{}, Conversation{}, err
	}
	body := strings.TrimSpace(request.Body)
	if body == "" {
		return CreateMessage{}, Conversation{}, ErrEmptyBody
	}
	if len(body) > MaxBodyLength {
		return CreateMessage{}, Conversation{}, ErrBodyTooLong
	}
	return CreateMessage{
		SenderID:    user.ID,
		RecipientID: request.RecipientID,
		EntryID:     request.EntryID,
		Body:        body,
		CreatedAt:   now.UTC(),
	}, conversation, nil
}

func (Policy) ValidateEntry(conversation Conversation, entryOfficerID int) error {
	if entryOfficerID != conversation.OfficerID {
		return ErrEntryNotFound
	}
	return nil
}

func (Policy) ValidateDelete(user User, message Message) error {
	if message.ID == 0 {
		return ErrMessageNotFound
	}
	if message.SenderID != user.ID {
		return ErrForbidden
	}
	return nil
}

func (Policy) UnreadMessageIDs(messages []Message, userID int) []int {
	var ids []int
	for _, message := range messages {
		if message.RecipientID == userID && message.ReadAt == nil {
			ids = append(ids, message.ID)
		}
	}
	return ids
}

func (Policy) ApplyReadAt(messages []Message, userID int, readAt time.Time) []Message {
	for i := range messages {
		if messages[i].RecipientID == userID && messages[i].ReadAt == nil {
			value := readAt.UTC()
			messages[i].ReadAt = &value
		}
	}
	return messages
}
