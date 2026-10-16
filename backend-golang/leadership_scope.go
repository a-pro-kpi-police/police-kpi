package main

import (
	"context"
	"encoding/json"
	"net/http"
)

type LeadershipScope struct {
	Mode        string `json:"mode"`
	DistrictIDs []int  `json:"district_ids"`
}

// Only legacy leadership rows are migrated. The explicit value makes restarts
// idempotent and decouples future affiliation changes from viewing permissions.
func migrateLeadershipScope(db *Store) error {
	_, err := db.ExecContext(context.Background(), `UPDATE users SET leadership_scope = CASE
 WHEN district_id IS NULL THEN '{"mode":"all","district_ids":[]}'
 ELSE '{"mode":"selected","district_ids":[' || CAST(district_id AS TEXT) || ']}' END
 WHERE role = 'leadership' AND leadership_scope = ''`)
	return err
}

func (s *LeadershipScope) allows(id int) bool {
	if s == nil {
		return false
	}
	if s.Mode == "all" {
		return true
	}
	if s.Mode != "selected" {
		return false
	}
	for _, allowed := range s.DistrictIDs {
		if allowed == id {
			return true
		}
	}
	return false
}

func scopeJSON(scope *LeadershipScope) string {
	if scope == nil {
		return ""
	}
	value, _ := json.Marshal(scope)
	return string(value)
}

func (a *App) prepareLeadershipScope(ctx context.Context, role string, scope *LeadershipScope) (*LeadershipScope, error) {
	if role != roleLeadership {
		return nil, nil
	}
	if scope == nil || (scope.Mode != "all" && scope.Mode != "selected") {
		return nil, httpErr{status: http.StatusBadRequest, detail: "Укажите доступ к ротам"}
	}
	result := &LeadershipScope{Mode: scope.Mode, DistrictIDs: []int{}}
	if scope.Mode == "all" {
		return result, nil
	}
	if len(scope.DistrictIDs) == 0 {
		return nil, httpErr{status: 400, detail: "Выберите хотя бы одну роту"}
	}
	seen := map[int]bool{}
	for _, id := range scope.DistrictIDs {
		if id <= 0 {
			return nil, httpErr{status: 400, detail: "Рота указана неверно"}
		}
		if err := a.ensureDistrictExists(ctx, id); err != nil {
			return nil, err
		}
		if !seen[id] {
			result.DistrictIDs = append(result.DistrictIDs, id)
			seen[id] = true
		}
	}
	return result, nil
}

func loadLeadershipScope(ctx context.Context, q dbtx, user *User) error {
	if user.Role != roleLeadership {
		return nil
	}
	var raw string
	if err := q.QueryRowContext(ctx, `SELECT leadership_scope FROM users WHERE id = ?`, user.ID).Scan(&raw); err != nil {
		return err
	}
	// Missing or invalid persisted settings fail closed, never fall back to affiliation.
	user.LeadershipScope = &LeadershipScope{Mode: "selected", DistrictIDs: []int{}}
	if raw == "" {
		return nil
	}
	return json.Unmarshal([]byte(raw), user.LeadershipScope)
}
