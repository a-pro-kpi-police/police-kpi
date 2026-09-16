package main

import "context"

// globalKPIAverage returns only an aggregate, never an unrestricted officer list.
// LEFT JOIN keeps officers without confirmed activity in the denominator.
func (a *App) globalKPIAverage(ctx context.Context, month string) (float64, error) {
	query := `SELECT COALESCE(AVG(score), 0) FROM (
		SELECT u.id, ROUND(CAST(COALESCE(SUM(e.final_score), 0) AS NUMERIC), 2) AS score
		FROM users u
		LEFT JOIN kpi_entries e ON e.officer_id = u.id
			AND e.status = 'confirmed' AND e.quantity > 0`
	args := []any{}
	if month != "" {
		query += ` AND e.month = ?`
		args = append(args, month)
	}
	query += ` WHERE u.role = ? GROUP BY u.id) officer_scores`
	args = append(args, roleOfficer)
	var average float64
	err := a.db.QueryRowContext(ctx, query, args...).Scan(&average)
	return average, err
}
