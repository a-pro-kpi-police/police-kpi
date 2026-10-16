package users

import (
	"context"
	"database/sql"
	"errors"

	domainusers "kpi-police/backend-golang/internal/domain/users"
	"kpi-police/backend-golang/internal/infrastructure/viewingscope"
)

type Queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Repository struct {
	q Queryer
}

func NewRepository(q Queryer) Repository {
	return Repository{q: q}
}

func (r Repository) GetByID(ctx context.Context, id int) (domainusers.User, error) {
	return r.queryUser(ctx, "u.id = ?", id)
}

func (r Repository) GetByUsername(ctx context.Context, username string) (domainusers.User, error) {
	return r.queryUser(ctx, "u.username = ?", username)
}

func (r Repository) GetOfficer(ctx context.Context, id int) (domainusers.User, error) {
	user, err := r.GetByID(ctx, id)
	if err != nil {
		return domainusers.User{}, err
	}
	if user.Role != domainusers.RoleOfficer {
		return domainusers.User{}, errors.Join(sql.ErrNoRows, domainusers.ErrNotFound)
	}
	return user, nil
}

func (r Repository) GetDistrict(ctx context.Context, id int) (domainusers.District, error) {
	var district domainusers.District
	err := r.q.QueryRowContext(ctx, `SELECT id, name, coeff FROM districts WHERE id = ?`, id).
		Scan(&district.ID, &district.Name, &district.Coeff)
	if err == sql.ErrNoRows {
		return domainusers.District{}, errors.Join(sql.ErrNoRows, domainusers.ErrNotFound)
	}
	if err != nil {
		return domainusers.District{}, err
	}
	return district, nil
}

func (r Repository) ListDistricts(ctx context.Context) ([]domainusers.District, error) {
	args := []any{}
	query := `SELECT id, name, coeff FROM districts WHERE ` + viewingscope.Predicate(ctx, "id", &args) + ` ORDER BY id`
	rows, err := r.q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var districts []domainusers.District
	for rows.Next() {
		var district domainusers.District
		if err := rows.Scan(&district.ID, &district.Name, &district.Coeff); err != nil {
			return nil, err
		}
		districts = append(districts, district)
	}
	return districts, rows.Err()
}

func (r Repository) ListByRole(ctx context.Context, role string) ([]domainusers.User, error) {
	return r.queryUsers(ctx, userSelect+` WHERE u.role = ? ORDER BY u.full_name`, role)
}

func (r Repository) ListOfficers(ctx context.Context, districtID int) ([]domainusers.User, error) {
	sqlText := userSelect + ` WHERE u.role = ?`
	args := []any{domainusers.RoleOfficer}
	if districtID > 0 {
		sqlText += ` AND u.district_id = ?`
		args = append(args, districtID)
	}
	sqlText += ` AND ` + viewingscope.Predicate(ctx, "u.district_id", &args)
	sqlText += ` ORDER BY u.full_name`
	return r.queryUsers(ctx, sqlText, args...)
}

func (r Repository) ConfirmedScores(ctx context.Context) (map[int]float64, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT officer_id, COALESCE(SUM(final_score), 0)
		FROM kpi_entries
		WHERE status = ?
		GROUP BY officer_id`, "confirmed")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	scores := map[int]float64{}
	for rows.Next() {
		var officerID int
		var score float64
		if err := rows.Scan(&officerID, &score); err != nil {
			return nil, err
		}
		scores[officerID] = score
	}
	return scores, rows.Err()
}

func (r Repository) ListMonths(ctx context.Context, officerID *int, districtID *int) ([]string, error) {
	sqlText := `SELECT month FROM (
		SELECT DISTINCT month FROM kpi_entries
		UNION
		SELECT DISTINCT month FROM kpi_targets
	) AS months ORDER BY month DESC`
	args := []any{}
	if officerID != nil {
		sqlText = `SELECT month FROM (
			SELECT DISTINCT month FROM kpi_entries WHERE officer_id = ?
			UNION
			SELECT DISTINCT month FROM kpi_targets WHERE officer_id = ?
		) AS months ORDER BY month DESC`
		args = append(args, *officerID, *officerID)
	} else if districtID != nil {
		sqlText = `SELECT month FROM (
			SELECT DISTINCT e.month FROM kpi_entries e
			JOIN users u ON u.id = e.officer_id
			WHERE u.district_id = ?
			UNION
			SELECT DISTINCT t.month FROM kpi_targets t
			JOIN users u ON u.id = t.officer_id
			WHERE u.district_id = ?
		) AS months ORDER BY month DESC`
		args = append(args, *districtID, *districtID)
	}
	scopeArgs := []any{}
	boundary := viewingscope.Predicate(ctx, "u.district_id", &scopeArgs)
	if boundary != "1=1" {
		sqlText = `SELECT month FROM (
        SELECT DISTINCT e.month FROM kpi_entries e JOIN users u ON u.id=e.officer_id WHERE ` + boundary + `
        UNION SELECT DISTINCT t.month FROM kpi_targets t JOIN users u ON u.id=t.officer_id WHERE ` + boundary + `
        ) AS months ORDER BY month DESC`
		args = append(scopeArgs, scopeArgs...)
	}
	rows, err := r.q.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	months := []string{}
	for rows.Next() {
		var month string
		if err := rows.Scan(&month); err != nil {
			return nil, err
		}
		months = append(months, month)
	}
	return months, rows.Err()
}

func (r Repository) queryUser(ctx context.Context, where string, args ...any) (domainusers.User, error) {
	user, err := scanUser(r.q.QueryRowContext(ctx, userSelect+` WHERE `+where+` LIMIT 1`, args...))
	if err == sql.ErrNoRows {
		return domainusers.User{}, errors.Join(sql.ErrNoRows, domainusers.ErrNotFound)
	}
	return user, err
}

func (r Repository) queryUsers(ctx context.Context, query string, args ...any) ([]domainusers.User, error) {
	rows, err := r.q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []domainusers.User
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func scanUser(scanner interface {
	Scan(dest ...any) error
}) (domainusers.User, error) {
	var user domainusers.User
	var districtID sql.NullInt64
	var sourceEmployeeNumber sql.NullString
	var joinedDistrictID sql.NullInt64
	var districtName sql.NullString
	var districtCoeff sql.NullFloat64
	if err := scanner.Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.FullName,
		&user.Rank,
		&user.Role,
		&districtID,
		&sourceEmployeeNumber,
		&user.MonthlyNorm,
		&joinedDistrictID,
		&districtName,
		&districtCoeff,
	); err != nil {
		return domainusers.User{}, err
	}
	if districtID.Valid {
		id := int(districtID.Int64)
		user.DistrictID = &id
	}
	if sourceEmployeeNumber.Valid {
		value := sourceEmployeeNumber.String
		user.SourceEmployeeNumber = &value
	}
	if joinedDistrictID.Valid {
		user.District = &domainusers.District{
			ID:    int(joinedDistrictID.Int64),
			Name:  districtName.String,
			Coeff: districtCoeff.Float64,
		}
	}
	return user, nil
}

const userSelect = `SELECT
	u.id, u.username, u.password_hash, u.full_name, u.rank, u.role,
	u.district_id, u.source_employee_number, u.monthly_norm,
	d.id, d.name, d.coeff
FROM users u
LEFT JOIN districts d ON d.id = u.district_id`
