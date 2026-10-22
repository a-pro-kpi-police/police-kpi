package users

import "errors"

const (
	RoleLeadership = "leadership"
	RoleAdmin      = "admin"
	RoleHead       = "head"
	RoleOfficer    = "officer"
)

var ErrNotFound = errors.New("user not found")

type District struct {
	ID    int
	Name  string
	Coeff float64
}

type User struct {
	ID                   int
	Username             string
	PasswordHash         string
	FullName             string
	Rank                 string
	Role                 string
	DistrictID           *int
	SourceEmployeeNumber *string
	MonthlyNorm          float64
	District             *District
}
