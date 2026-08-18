package catalog

import (
	"context"
	"database/sql"
	"strings"

	domaincatalog "kpi-police/backend-golang/internal/domain/catalog"
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

func (r Repository) Get(ctx context.Context, id int) (domaincatalog.Item, error) {
	return r.queryItem(ctx, catalogSelect+` WHERE id = ?`, id)
}

func (r Repository) ListActive(ctx context.Context, includeNegative bool) ([]domaincatalog.Item, error) {
	where := []string{"is_active = 1"}
	if !includeNegative {
		where = append(where, "points_per_unit >= 0")
	}
	return r.queryItems(ctx, catalogSelect+` WHERE `+strings.Join(where, " AND ")+` ORDER BY block, code`)
}

func (r Repository) ListActivePositive(ctx context.Context) ([]domaincatalog.Item, error) {
	return r.queryItems(ctx, catalogSelect+` WHERE is_active = 1 AND points_per_unit >= 0 ORDER BY block, code`)
}

func (r Repository) ListAll(ctx context.Context) ([]domaincatalog.Item, error) {
	return r.queryItems(ctx, catalogSelect+` ORDER BY block, code`)
}

func (r Repository) queryItem(ctx context.Context, query string, args ...any) (domaincatalog.Item, error) {
	return scanItem(r.q.QueryRowContext(ctx, query, args...))
}

func (r Repository) queryItems(ctx context.Context, query string, args ...any) ([]domaincatalog.Item, error) {
	rows, err := r.q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domaincatalog.Item
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanItem(scanner interface {
	Scan(dest ...any) error
}) (domaincatalog.Item, error) {
	var item domaincatalog.Item
	var isMass, isCritical, isActive int
	err := scanner.Scan(
		&item.ID,
		&item.Code,
		&item.Indicator,
		&item.Block,
		&item.Subblock,
		&item.NPA,
		&item.PointsPerUnit,
		&item.Unit,
		&item.SourceURL,
		&item.MonthlyLimit,
		&isMass,
		&isCritical,
		&isActive,
	)
	item.IsMass = isMass != 0
	item.IsCritical = isCritical != 0
	item.IsActive = isActive != 0
	return item, err
}

const catalogSelect = `SELECT
	id, code, indicator, block, subblock, npa, points_per_unit, unit,
	source_url, monthly_limit, is_mass, is_critical, is_active
FROM kpi_catalog`
