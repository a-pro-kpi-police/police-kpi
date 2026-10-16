package dashboard

import (
	"context"
	"database/sql"
	"strings"

	domaindashboard "kpi-police/backend-golang/internal/domain/dashboard"
	"kpi-police/backend-golang/internal/infrastructure/viewingscope"
)

type Queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type Repository struct {
	q Queryer
}

func NewRepository(q Queryer) Repository {
	return Repository{q: q}
}

func (r Repository) ListEntries(ctx context.Context, filter domaindashboard.Filter) ([]domaindashboard.Entry, error) {
	args := []any{}
	where := []string{viewingscope.Predicate(ctx, "u.district_id", &args)}
	if filter.OfficerID != nil {
		where = append(where, "e.officer_id = ?")
		args = append(args, *filter.OfficerID)
	}
	if filter.DistrictID != nil {
		where = append(where, "u.district_id = ?")
		args = append(args, *filter.DistrictID)
	}
	if filter.Month != "" {
		where = append(where, "e.month = ?")
		args = append(args, filter.Month)
	}

	query := dashboardEntrySelect
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	rows, err := r.q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []domaindashboard.Entry
	for rows.Next() {
		entry, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (r Repository) ListCatalogBlocks(ctx context.Context) ([]string, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT block FROM kpi_catalog WHERE is_active = 1 ORDER BY block, code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seen := map[string]bool{}
	var blocks []string
	for rows.Next() {
		var block string
		if err := rows.Scan(&block); err != nil {
			return nil, err
		}
		if block != "" && !seen[block] {
			seen[block] = true
			blocks = append(blocks, block)
		}
	}
	return blocks, rows.Err()
}

func scanEntry(scanner interface {
	Scan(dest ...any) error
}) (domaindashboard.Entry, error) {
	var entry domaindashboard.Entry
	var officerName, officerRank, officerDistrict sql.NullString
	var catalogCode, catalogIndicator, catalogBlock sql.NullString
	var isCritical int
	err := scanner.Scan(
		&entry.Month,
		&entry.OfficerID,
		&entry.Quantity,
		&entry.CountedQuantity,
		&entry.PointsPerUnit,
		&entry.RawScore,
		&entry.FinalScore,
		&entry.Status,
		&officerName,
		&officerRank,
		&officerDistrict,
		&catalogCode,
		&catalogIndicator,
		&catalogBlock,
		&isCritical,
	)
	entry.OfficerName = officerName.String
	entry.OfficerRank = officerRank.String
	entry.OfficerDistrict = officerDistrict.String
	entry.CatalogCode = catalogCode.String
	entry.CatalogIndicator = catalogIndicator.String
	entry.CatalogBlock = catalogBlock.String
	entry.IsCritical = isCritical != 0
	return entry, err
}

const dashboardEntrySelect = `SELECT
	e.month, e.officer_id, e.quantity, e.counted_quantity, e.points_per_unit, e.raw_score, e.final_score, e.status,
	u.full_name, u.rank, d.name,
	c.code, c.indicator, c.block, c.is_critical
FROM kpi_entries e
LEFT JOIN users u ON u.id = e.officer_id
LEFT JOIN districts d ON d.id = u.district_id
LEFT JOIN kpi_catalog c ON c.id = e.catalog_id`
