package notifications

import (
	"errors"
	"time"
)

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

var (
	ErrInvalidLimit = errors.New("invalid notification limit")
	ErrNotFound     = errors.New("notification not found")
)

type Notification struct {
	ID        int
	Kind      string
	Title     string
	Message   string
	Link      string
	IsRead    bool
	CreatedAt time.Time
}

type List struct {
	UnreadCount int
	Items       []Notification
}

type Policy struct{}

func NewPolicy() Policy {
	return Policy{}
}

func (Policy) ResolveLimit(limit int) (int, error) {
	if limit == 0 {
		return DefaultLimit, nil
	}
	if limit < 1 || limit > MaxLimit {
		return 0, ErrInvalidLimit
	}
	return limit, nil
}
