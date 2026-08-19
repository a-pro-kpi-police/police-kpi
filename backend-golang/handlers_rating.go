package main

import (
	"context"
	"database/sql"
	"net/http"
	"sort"

	domainusers "kpi-police/backend-golang/internal/domain/users"
)

type ratingScore struct {
	Final     float64
	Confirmed int
	Pending   int
}

func (a *App) handleRating(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireHead(a, w, r); !ok {
		return
	}
	month := r.URL.Query().Get("month")
	officers, err := a.userQueryService().ListOfficersWithEfficiency(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	scores, err := a.ratingScores(r.Context(), month)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ratingOfficersOut(officers, scores))
}

func (a *App) ratingScores(ctx context.Context, month string) (map[int]ratingScore, error) {
	query := `SELECT officer_id,
		COALESCE(SUM(CASE WHEN status = ? AND quantity > 0 THEN final_score ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN status = ? AND quantity > 0 THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN status = ? AND quantity > 0 THEN 1 ELSE 0 END), 0)
	FROM kpi_entries`
	args := []any{"confirmed", "confirmed", "pending"}
	if month != "" {
		query += ` WHERE month = ?`
		args = append(args, month)
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
