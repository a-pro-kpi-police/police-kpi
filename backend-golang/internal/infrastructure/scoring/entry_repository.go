package scoring

import (
	"context"
	"database/sql"

	domainscoring "kpi-police/backend-golang/internal/domain/scoring"
)

type DBTX interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type EntryRepository struct {
	q DBTX
}

func NewEntryRepository(q DBTX) EntryRepository {
	return EntryRepository{q: q}
}

func (r EntryRepository) ListOfficerMonth(ctx context.Context, officerID int, month string) ([]domainscoring.Entry, error) {
	rows, err := r.q.QueryContext(ctx, `SELECT
		e.id, e.catalog_id,
		e.quantity, e.counted_quantity, e.points_per_unit, e.coeff,
		e.raw_score, e.capped_score, e.anti_coeff, e.final_score,
		e.status,
		c.monthly_limit, c.is_mass, c.is_critical
	FROM kpi_entries e
	LEFT JOIN kpi_catalog c ON c.id = e.catalog_id
	WHERE e.officer_id = ? AND e.month = ?
	ORDER BY e.date, e.id`, officerID, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []domainscoring.Entry
	for rows.Next() {
		entry, err := scanScoringEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (r EntryRepository) SaveScoredEntries(ctx context.Context, entries []domainscoring.Entry) error {
	for _, entry := range entries {
		if _, err := r.q.ExecContext(ctx, `UPDATE kpi_entries SET
			raw_score = ?, counted_quantity = ?, capped_score = ?, coeff = ?, anti_coeff = ?, final_score = ?
			WHERE id = ?`,
			entry.RawScore,
			entry.CountedQuantity,
			entry.CappedScore,
			entry.Coeff,
			entry.AntiCoeff,
			entry.FinalScore,
			entry.ID,
		); err != nil {
			return err
		}
	}
	return nil
}

func scanScoringEntry(rows *sql.Rows) (domainscoring.Entry, error) {
	var entry domainscoring.Entry
	var monthlyLimit sql.NullFloat64
	var isMass, isCritical sql.NullInt64
	err := rows.Scan(
		&entry.ID,
		&entry.CatalogID,
		&entry.Quantity,
		&entry.CountedQuantity,
		&entry.PointsPerUnit,
		&entry.Coeff,
		&entry.RawScore,
		&entry.CappedScore,
		&entry.AntiCoeff,
		&entry.FinalScore,
		&entry.Status,
		&monthlyLimit,
		&isMass,
		&isCritical,
	)
	if err != nil {
		return domainscoring.Entry{}, err
	}
	if monthlyLimit.Valid || isMass.Valid || isCritical.Valid {
		entry.Catalog = &domainscoring.CatalogSnapshot{
			MonthlyLimit: monthlyLimit.Float64,
			IsMass:       isMass.Valid && isMass.Int64 != 0,
			IsCritical:   isCritical.Valid && isCritical.Int64 != 0,
		}
	}
	return entry, nil
}
