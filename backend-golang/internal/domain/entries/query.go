package entries

import "errors"

var (
	ErrInvalidLimit  = errors.New("invalid limit")
	ErrInvalidOffset = errors.New("invalid offset")
)

type EntryQuery struct {
	Actor         Actor
	OfficerID     *int
	DistrictID    *int
	Month         string
	Months        []string
	Year          string
	Status        string
	SortKey       string
	SortAscending bool
	Limit         *int
	Offset        *int
}

type ListedEntry struct {
	ID                    int
	Date                  string
	Month                 string
	OfficerID             int
	OfficerName           string
	Rank                  string
	DistrictName          string
	CatalogID             int
	Code                  string
	Indicator             string
	Block                 string
	Subblock              string
	NPA                   string
	SourceURL             string
	Quantity              float64
	CountedQuantity       float64
	MonthlyLimit          float64
	IsMass                bool
	IsCritical            bool
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

type QueryPolicy struct{}

func NewQueryPolicy() QueryPolicy {
	return QueryPolicy{}
}

func (QueryPolicy) Normalize(query EntryQuery) (EntryQuery, error) {
	if query.Actor.Role != RoleHead && query.Actor.Role != RoleAdmin {
		officerID := query.Actor.ID
		query.OfficerID = &officerID
	}
	if query.Limit != nil && (*query.Limit < 1 || *query.Limit > 100) {
		return EntryQuery{}, ErrInvalidLimit
	}
	if query.Offset != nil && *query.Offset < 0 {
		return EntryQuery{}, ErrInvalidOffset
	}
	return query, nil
}
