package main

import (
	"context"
	"database/sql"
	"net/http"
)

func scopedHeadDistrictID(user *User, requested *int) (*int, error) {
	if user != nil && user.Role == roleLeadership {
		if requested != nil && !user.LeadershipScope.allows(*requested) {
			return nil, httpErr{status: 403, detail: "Недостаточно прав для другой роты"}
		}
		return requested, nil
	}
	if user == nil || user.Role != roleHead || !user.DistrictID.Valid {
		return requested, nil
	}
	scope := int(user.DistrictID.Int64)
	if requested != nil && *requested != scope {
		return nil, httpErr{status: http.StatusForbidden, detail: "Недостаточно прав для другой роты"}
	}
	return &scope, nil
}

func headScopeDistrictValue(user *User) (int, bool) {
	if user == nil || user.Role != roleHead || !user.DistrictID.Valid {
		return 0, false
	}
	return int(user.DistrictID.Int64), true
}

func (a *App) ensureOfficerInHeadScope(ctx context.Context, user *User, officerID int) error {
	scope, ok := headScopeDistrictValue(user)
	if !ok && (user == nil || user.Role != roleLeadership) {
		return nil
	}
	var districtID sql.NullInt64
	err := a.db.QueryRowContext(ctx, `SELECT district_id FROM users WHERE id = ? AND role = ?`, officerID, roleOfficer).
		Scan(&districtID)
	if err == sql.ErrNoRows {
		return httpErr{status: http.StatusNotFound, detail: "Сотрудник не найден"}
	}
	if err != nil {
		return err
	}
	if user.Role == roleLeadership && user.LeadershipScope != nil && user.LeadershipScope.Mode == "all" {
		return nil
	}
	allowed := districtID.Valid && int(districtID.Int64) == scope
	if user.Role == roleLeadership {
		allowed = districtID.Valid && user.LeadershipScope.allows(int(districtID.Int64))
	}
	if !allowed {
		return httpErr{status: http.StatusForbidden, detail: "Недостаточно прав для другой роты"}
	}
	return nil
}

func optionalIntValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
