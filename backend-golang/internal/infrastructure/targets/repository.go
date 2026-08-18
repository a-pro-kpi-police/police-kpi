package targets

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	apptargets "kpi-police/backend-golang/internal/application/targets"
	domaintargets "kpi-police/backend-golang/internal/domain/targets"
)

type Queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Tx interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	Commit() error
	Rollback() error
}

type BeginTxFunc func(context.Context, *sql.TxOptions) (Tx, error)

type Repository struct {
	q       Queryer
	beginTx BeginTxFunc
}

func NewRepository(q Queryer, beginTx BeginTxFunc) Repository {
	return Repository{q: q, beginTx: beginTx}
}

func (r Repository) GetOfficer(ctx context.Context, id int) (apptargets.Officer, error) {
	var officerID int
	var role string
	err := r.q.QueryRowContext(ctx, `SELECT id, role FROM users WHERE id = ?`, id).Scan(&officerID, &role)
	if err == sql.ErrNoRows {
		return apptargets.Officer{}, domaintargets.ErrOfficerNotFound
	}
	if err != nil {
		return apptargets.Officer{}, err
	}
	if role != "officer" {
		return apptargets.Officer{}, domaintargets.ErrOfficerNotFound
	}
	return apptargets.Officer{ID: officerID}, nil
}

func (r Repository) DistrictOfficerIDs(ctx context.Context, districtID int) ([]int, error) {
	var exists int
	if err := r.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM districts WHERE id = ?`, districtID).Scan(&exists); err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, domaintargets.ErrDistrictNotFound
	}
	rows, err := r.q.QueryContext(ctx, `SELECT id FROM users WHERE role = ? AND district_id = ? ORDER BY full_name`, "officer", districtID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var officerIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		officerIDs = append(officerIDs, id)
	}
	return officerIDs, rows.Err()
}

func (r Repository) ActivePositiveCatalogIDs(ctx context.Context) (map[int]bool, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT id FROM kpi_catalog WHERE is_active = 1 AND points_per_unit >= 0`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	valid := map[int]bool{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		valid[id] = true
	}
	return valid, rows.Err()
}

func (r Repository) GetOfficerProfile(ctx context.Context, id int) (domaintargets.Officer, error) {
	officer, err := scanOfficer(r.q.QueryRowContext(ctx, officerSelect+` WHERE u.id = ? AND u.role = ?`, id, "officer"))
	if err == sql.ErrNoRows {
		return domaintargets.Officer{}, domaintargets.ErrOfficerNotFound
	}
	return officer, err
}

func (r Repository) GetDistrict(ctx context.Context, id int) (domaintargets.District, error) {
	var district domaintargets.District
	err := r.q.QueryRowContext(ctx, `SELECT id, name, coeff FROM districts WHERE id = ?`, id).
		Scan(&district.ID, &district.Name, &district.Coeff)
	if err == sql.ErrNoRows {
		return domaintargets.District{}, domaintargets.ErrDistrictNotFound
	}
	if err != nil {
		return domaintargets.District{}, err
	}
	return district, nil
}

func (r Repository) ListOfficerProfiles(ctx context.Context, districtID int) ([]domaintargets.Officer, error) {
	sqlText := officerSelect + ` WHERE u.role = ?`
	args := []any{"officer"}
	if districtID > 0 {
		sqlText += ` AND u.district_id = ?`
		args = append(args, districtID)
	}
	sqlText += ` ORDER BY u.full_name`
	rows, err := r.q.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var officers []domaintargets.Officer
	for rows.Next() {
		officer, err := scanOfficer(rows)
		if err != nil {
			return nil, err
		}
		officers = append(officers, officer)
	}
	return officers, rows.Err()
}

func (r Repository) ListActivePositiveCatalog(ctx context.Context) ([]domaintargets.CatalogItem, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT id, code, indicator, block, unit,
			points_per_unit, monthly_limit, is_mass, is_critical
		FROM kpi_catalog
		WHERE is_active = 1 AND points_per_unit >= 0
		ORDER BY block, code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domaintargets.CatalogItem
	for rows.Next() {
		var item domaintargets.CatalogItem
		var isMass, isCritical int
		if err := rows.Scan(
			&item.CatalogID,
			&item.Code,
			&item.Indicator,
			&item.Block,
			&item.Unit,
			&item.PointsPerUnit,
			&item.MonthlyLimit,
			&isMass,
			&isCritical,
		); err != nil {
			return nil, err
		}
		item.IsMass = isMass != 0
		item.IsCritical = isCritical != 0
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r Repository) OfficerTargetQuantities(ctx context.Context, officerID int, month string) (map[int]float64, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT catalog_id, target_quantity FROM kpi_targets WHERE officer_id = ? AND month = ?`, officerID, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	quantities := map[int]float64{}
	for rows.Next() {
		var catalogID int
		var quantity float64
		if err := rows.Scan(&catalogID, &quantity); err != nil {
			return nil, err
		}
		quantities[catalogID] = quantity
	}
	return quantities, rows.Err()
}

func (r Repository) GroupTargetQuantities(ctx context.Context, officerIDs []int, month string) (map[int]map[int]float64, error) {
	quantities := map[int]map[int]float64{}
	if len(officerIDs) == 0 {
		return quantities, nil
	}
	args := []any{month}
	for _, id := range officerIDs {
		args = append(args, id)
	}
	rows, err := r.q.QueryContext(ctx, `SELECT officer_id, catalog_id, target_quantity
		FROM kpi_targets
		WHERE month = ? AND officer_id IN (`+placeholders(len(officerIDs))+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var officerID, catalogID int
		var quantity float64
		if err := rows.Scan(&officerID, &catalogID, &quantity); err != nil {
			return nil, err
		}
		if quantities[officerID] == nil {
			quantities[officerID] = map[int]float64{}
		}
		quantities[officerID][catalogID] = quantity
	}
	return quantities, rows.Err()
}

func (r Repository) ActualsByCatalog(ctx context.Context, officerID int, month string) (map[int]domaintargets.Actual, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT catalog_id, SUM(quantity), SUM(counted_quantity), SUM(final_score)
		FROM kpi_entries
		WHERE officer_id = ? AND month = ? AND status = ?
		GROUP BY catalog_id`, officerID, month, "confirmed")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	actuals := map[int]domaintargets.Actual{}
	for rows.Next() {
		var catalogID int
		var quantity, counted, score sql.NullFloat64
		if err := rows.Scan(&catalogID, &quantity, &counted, &score); err != nil {
			return nil, err
		}
		actuals[catalogID] = domaintargets.Actual{
			Quantity:        quantity.Float64,
			CountedQuantity: counted.Float64,
			Score:           score.Float64,
		}
	}
	return actuals, rows.Err()
}

func (r Repository) PenaltiesByOfficer(ctx context.Context, month string, officerIDs []int) (map[int]domaintargets.Penalty, error) {
	penalties := map[int]domaintargets.Penalty{}
	if len(officerIDs) == 0 {
		return penalties, nil
	}
	args := []any{month, "confirmed"}
	for _, id := range officerIDs {
		args = append(args, id)
	}
	rows, err := r.q.QueryContext(ctx, `SELECT officer_id, SUM(final_score), COUNT(*)
		FROM kpi_entries
		WHERE month = ? AND status = ? AND points_per_unit < 0 AND quantity > 0
			AND officer_id IN (`+placeholders(len(officerIDs))+`)
		GROUP BY officer_id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var officerID int
		var score sql.NullFloat64
		var count int
		if err := rows.Scan(&officerID, &score, &count); err != nil {
			return nil, err
		}
		penalties[officerID] = domaintargets.Penalty{Score: score.Float64, Count: count}
	}
	return penalties, rows.Err()
}

func (r Repository) SaveOfficerTargets(ctx context.Context, officerID int, month string, quantities map[int]float64) error {
	tx, err := r.beginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := applyTargetQuantities(ctx, tx, []int{officerID}, month, quantities, true); err != nil {
		return err
	}
	title, message := targetUpdatedNotification(month)
	if err := addNotification(ctx, tx, officerID, "targets_updated", title, message, "/targets"); err != nil {
		return err
	}
	return tx.Commit()
}

func (r Repository) SaveGroupTargets(ctx context.Context, officerIDs []int, month string, quantities map[int]float64) error {
	tx, err := r.beginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := applyTargetQuantities(ctx, tx, officerIDs, month, quantities, false); err != nil {
		return err
	}
	title, message := targetUpdatedNotification(month)
	for _, officerID := range officerIDs {
		if err := addNotification(ctx, tx, officerID, "targets_updated", title, message, "/targets"); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func applyTargetQuantities(ctx context.Context, q Tx, officerIDs []int, month string, quantities map[int]float64, replace bool) error {
	if len(officerIDs) == 0 {
		return nil
	}
	args := []any{month}
	for _, id := range officerIDs {
		args = append(args, id)
	}
	rows, err := q.QueryContext(ctx, `SELECT id, officer_id, catalog_id FROM kpi_targets WHERE month = ? AND officer_id IN (`+placeholders(len(officerIDs))+`)`, args...)
	if err != nil {
		return err
	}
	type existingTarget struct {
		ID        int
		OfficerID int
		CatalogID int
	}
	var existing []existingTarget
	for rows.Next() {
		var item existingTarget
		if err := rows.Scan(&item.ID, &item.OfficerID, &item.CatalogID); err != nil {
			rows.Close()
			return err
		}
		existing = append(existing, item)
	}
	rows.Close()
	for _, target := range existing {
		quantity, inPayload := quantities[target.CatalogID]
		shouldDelete := (replace && !inPayload) || (inPayload && quantity <= 0)
		if shouldDelete {
			if _, err := q.ExecContext(ctx, `DELETE FROM kpi_targets WHERE id = ?`, target.ID); err != nil {
				return err
			}
		}
	}
	for _, officerID := range officerIDs {
		for catalogID, quantity := range quantities {
			if quantity <= 0 {
				continue
			}
			if _, err := q.ExecContext(ctx, `INSERT INTO kpi_targets(month, officer_id, catalog_id, target_quantity)
				VALUES (?, ?, ?, ?)
				ON CONFLICT(month, officer_id, catalog_id) DO UPDATE SET target_quantity = excluded.target_quantity`,
				month, officerID, catalogID, quantity,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func addNotification(ctx context.Context, q Tx, userID int, kind string, title string, message string, link string) error {
	_, err := q.ExecContext(ctx, `INSERT INTO app_notifications
		(user_id, kind, title, message, link, is_read, created_at)
		VALUES (?, ?, ?, ?, ?, 0, ?)`,
		userID, kind, title, message, link, time.Now().UTC().Format(time.RFC3339Nano),
	)
	return err
}

func targetUpdatedNotification(month string) (string, string) {
	return "Обновлены целевые KPI", fmt.Sprintf("Начальник установил целевые показатели на %s.", month)
}

func scanOfficer(scanner interface {
	Scan(dest ...any) error
}) (domaintargets.Officer, error) {
	var officer domaintargets.Officer
	var districtID sql.NullInt64
	var districtName sql.NullString
	var districtCoeff sql.NullFloat64
	if err := scanner.Scan(
		&officer.ID,
		&officer.FullName,
		&officer.MonthlyNorm,
		&districtID,
		&districtName,
		&districtCoeff,
	); err != nil {
		return domaintargets.Officer{}, err
	}
	if districtID.Valid {
		officer.District = &domaintargets.District{
			ID:    int(districtID.Int64),
			Name:  districtName.String,
			Coeff: districtCoeff.Float64,
		}
	}
	return officer, nil
}

const officerSelect = `SELECT
	u.id, u.full_name, u.monthly_norm,
	d.id, d.name, d.coeff
FROM users u
LEFT JOIN districts d ON d.id = u.district_id`

func placeholders(count int) string {
	if count <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", count), ",")
}
