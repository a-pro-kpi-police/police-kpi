package entries

import (
	"errors"
	"math"
	"strings"
	"time"
)

const (
	RoleAdmin   = "admin"
	RoleHead    = "head"
	RoleOfficer = "officer"

	StatusPending   = "pending"
	StatusConfirmed = "confirmed"
	StatusRejected  = "rejected"
)

var (
	ErrOfficerRequired      = errors.New("officer required")
	ErrOfficerNotFound      = errors.New("officer not found")
	ErrCatalogNotFound      = errors.New("catalog not found")
	ErrEntryNotFound        = errors.New("entry not found")
	ErrForbidden            = errors.New("forbidden")
	ErrNegativeKPIForbidden = errors.New("negative kpi forbidden")
	ErrQuantityPositive     = errors.New("quantity must be positive")
	ErrInvalidDate          = errors.New("invalid date")
	ErrInvalidStatus        = errors.New("invalid status")
)

type Actor struct {
	ID       int
	FullName string
	Role     string
}

type Officer struct {
	ID    int
	Coeff float64
}

type Catalog struct {
	ID            int
	Code          string
	PointsPerUnit float64
	MonthlyLimit  float64
	IsActive      bool
}

type Attachment struct {
	Name        string
	StorageName string
	ContentType string
	Size        int64
}

type CreateRequest struct {
	CatalogID  int
	OfficerID  *int
	Quantity   *float64
	Date       *string
	Comments   string
	Attachment Attachment
}

type StatusUpdateRequest struct {
	EntryID     int
	Status      string
	HeadComment string
}

type Entry struct {
	ID                    int
	Date                  string
	Month                 string
	OfficerID             int
	CatalogID             int
	CatalogCode           string
	Quantity              float64
	CountedQuantity       float64
	PointsPerUnit         float64
	Coeff                 float64
	RawScore              float64
	CappedScore           float64
	AntiCoeff             float64
	FinalScore            float64
	Status                string
	Comments              string
	HeadComment           string
	AttachmentName        string
	AttachmentStorageName string
	AttachmentContentType string
	AttachmentSize        int64
}

type StatusChange struct {
	Entry          Entry
	PreviousStatus string
	NotifyOfficer  bool
}

type CreationPolicy struct{}

func NewCreationPolicy() CreationPolicy {
	return CreationPolicy{}
}

func (CreationPolicy) ResolveOfficerID(actor Actor, requestedOfficerID *int) (int, error) {
	switch actor.Role {
	case RoleHead, RoleAdmin:
		if requestedOfficerID == nil {
			return 0, ErrOfficerRequired
		}
		return *requestedOfficerID, nil
	case RoleOfficer:
		if requestedOfficerID != nil && *requestedOfficerID != actor.ID {
			return 0, ErrForbidden
		}
		return actor.ID, nil
	default:
		return 0, ErrForbidden
	}
}

func (CreationPolicy) BuildPendingEntry(request CreateRequest, actor Actor, officer Officer, catalog Catalog, defaultDate string) (Entry, error) {
	if !catalog.IsActive {
		return Entry{}, ErrCatalogNotFound
	}
	if !canManageEntries(actor.Role) && catalog.PointsPerUnit < 0 {
		return Entry{}, ErrNegativeKPIForbidden
	}

	quantity := 1.0
	if request.Quantity != nil {
		quantity = *request.Quantity
	}
	if quantity <= 0 {
		return Entry{}, ErrQuantityPositive
	}

	entryDate := strings.TrimSpace(defaultDate)
	if request.Date != nil {
		entryDate = strings.TrimSpace(*request.Date)
	}
	if !validDate(entryDate) {
		return Entry{}, ErrInvalidDate
	}

	counted := quantity
	if catalog.MonthlyLimit > 0 {
		counted = math.Min(counted, catalog.MonthlyLimit)
	}
	raw := catalog.PointsPerUnit * quantity
	capped := catalog.PointsPerUnit * counted
	attachment := request.Attachment

	return Entry{
		Date:                  entryDate,
		Month:                 monthFromDate(entryDate),
		OfficerID:             officer.ID,
		CatalogID:             catalog.ID,
		CatalogCode:           catalog.Code,
		Quantity:              quantity,
		CountedQuantity:       counted,
		PointsPerUnit:         catalog.PointsPerUnit,
		Coeff:                 officer.Coeff,
		RawScore:              raw,
		CappedScore:           capped,
		AntiCoeff:             1.0,
		FinalScore:            round2(capped * officer.Coeff),
		Status:                StatusPending,
		Comments:              request.Comments,
		HeadComment:           "",
		AttachmentName:        attachment.Name,
		AttachmentStorageName: attachment.StorageName,
		AttachmentContentType: attachment.ContentType,
		AttachmentSize:        attachment.Size,
	}, nil
}

func validDate(value string) bool {
	if len(value) != 10 {
		return false
	}
	_, err := time.Parse("2006-01-02", value)
	return err == nil
}

func monthFromDate(date string) string {
	if len(date) >= 7 {
		return date[:7]
	}
	return ""
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

type StatusPolicy struct{}

func NewStatusPolicy() StatusPolicy {
	return StatusPolicy{}
}

func (StatusPolicy) ChangeStatus(actor Actor, entry Entry, request StatusUpdateRequest) (StatusChange, error) {
	if !canManageEntries(actor.Role) {
		return StatusChange{}, ErrForbidden
	}
	status := strings.TrimSpace(request.Status)
	if !validStatus(status) {
		return StatusChange{}, ErrInvalidStatus
	}
	updated := entry
	updated.Status = status
	updated.HeadComment = request.HeadComment
	return StatusChange{
		Entry:          updated,
		PreviousStatus: entry.Status,
		NotifyOfficer:  shouldNotifyOfficer(entry.Status, status),
	}, nil
}

func validStatus(status string) bool {
	return status == StatusPending || status == StatusConfirmed || status == StatusRejected
}

func shouldNotifyOfficer(previousStatus string, nextStatus string) bool {
	return previousStatus != nextStatus && (nextStatus == StatusConfirmed || nextStatus == StatusRejected)
}

func canManageEntries(role string) bool {
	return role == RoleHead || role == RoleAdmin
}
