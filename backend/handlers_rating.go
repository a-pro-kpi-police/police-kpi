package main

import (
	"context"
	"database/sql"
	"net/http"
	"sort"
	"strings"

	domainusers "kpi-police/backend/internal/domain/users"
)

type ratingScore struct {
	Final     float64
	Confirmed int
	Pending   int
}

func (a *App) handleRating(w http.ResponseWriter, r *http.Request) {
	user, ok := requireHeadViewer(a, w, r)
	if !ok {
		return
	}
	scope, _ := headScopeDistrictValue(user)
	a.writeRating(w, r, scope, leadershipAverageScope(user))
}

func (a *App) handleAdminRating(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAdmin(a, w, r); !ok {
		return
	}
	districtID, err := parseOptionalInt(r.URL.Query().Get("district_id"))
	if err != nil || (districtID != nil && *districtID <= 0) {
		writeError(w, http.StatusBadRequest, "Участок указан неверно")
		return
	}
	if districtID != nil {
		if err := a.ensureDistrictExists(r.Context(), *districtID); err != nil {
			writeHTTPError(w, err)
			return
		}
	}
	a.writeRating(w, r, optionalIntValue(districtID), 0)
}

func (a *App) writeRating(w http.ResponseWriter, r *http.Request, scope, averageScope int) {
	month := r.URL.Query().Get("month")
	officers, err := a.userQueryService().ListOfficersWithEfficiency(r.Context(), scope)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	scores, err := a.ratingScores(r.Context(), month, scope)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	average, err := a.globalKPIAverage(r.Context(), month, averageScope)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	rows := ratingOfficersOut(officers, scores)
	for i := range rows {
		rows[i].GlobalAverageScore = average
	}
	writeJSON(w, http.StatusOK, rows)
}

func (a *App) ratingScores(ctx context.Context, month string, districtID int) (map[int]ratingScore, error) {
	query := `SELECT officer_id,
		COALESCE(SUM(CASE WHEN status = ? AND quantity > 0 THEN final_score ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN status = ? AND quantity > 0 THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN status = ? AND quantity > 0 THEN 1 ELSE 0 END), 0)
	FROM kpi_entries`
	args := []any{"confirmed", "confirmed", "pending"}
	where := []string{}
	if month != "" {
		where = append(where, "month = ?")
		args = append(args, month)
	}
	if districtID > 0 {
		query += ` JOIN users u ON u.id = kpi_entries.officer_id`
		where = append(where, "u.district_id = ?")
		args = append(args, districtID)
	}
	if len(where) > 0 {
		query += ` WHERE ` + strings.Join(where, " AND ")
	}
	query += ` GROUP BY officer_id`

	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	scores := map[int]ratingScore{}
	for rows.Next() {
		var officerID int
		var final sql.NullFloat64
		var confirmed, pending sql.NullInt64
		if err := rows.Scan(&officerID, &final, &confirmed, &pending); err != nil {
			return nil, err
		}
		scores[officerID] = ratingScore{
			Final:     round2(final.Float64),
			Confirmed: int(confirmed.Int64),
			Pending:   int(pending.Int64),
		}
	}
	return scores, rows.Err()
}

func ratingOfficersOut(officers []domainusers.OfficerWithEfficiency, scores map[int]ratingScore) []RatingOfficerOut {
	result := make([]RatingOfficerOut, 0, len(officers))
	for _, officer := range officers {
		user := officer.User
		score := scores[user.ID]
		row := RatingOfficerOut{
			OfficerID:       user.ID,
			Name:            user.FullName,
			Rank:            user.Rank,
			Final:           score.Final,
			Confirmed:       score.Confirmed,
			Pending:         score.Pending,
			CategoryScore:   score.Final,
			EfficiencyRank:  officer.Efficiency.Rank,
			EfficiencyTotal: officer.Efficiency.Total,
			EfficiencyScore: officer.Efficiency.Score,
		}
		if user.DistrictID != nil {
			row.DistrictID = *user.DistrictID
		}
		if user.District != nil {
			row.District = user.District.Name
		}
		result = append(result, row)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Final != result[j].Final {
			return result[i].Final > result[j].Final
		}
		return result[i].Name < result[j].Name
	})
	return result
}
