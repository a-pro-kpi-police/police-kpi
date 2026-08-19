package entries

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	domainentries "kpi-police/backend-golang/internal/domain/entries"
)

const listedEntrySelect = `SELECT
	e.id, e.date, e.month, e.officer_id, e.catalog_id,
	e.quantity, e.counted_quantity, e.points_per_unit, e.coeff,
	e.raw_score, e.capped_score, e.anti_coeff, e.final_score,
	e.status, e.comments, e.head_comment,
	e.attachment_name, e.attachment_storage_name, e.attachment_content_type, e.attachment_size,
	u.full_name, u.rank, d.name,
	c.code, c.indicator, c.block, c.subblock, c.npa, c.source_url,
	c.monthly_limit, c.is_mass, c.is_critical
FROM kpi_entries e
LEFT JOIN users u ON u.id = e.officer_id
LEFT JOIN districts d ON d.id = u.district_id
LEFT JOIN kpi_catalog c ON c.id = e.catalog_id`

type rowScanner interface {
	Scan(dest ...any) error
}

func (r Repository) ListEntries(ctx context.Context, query domainentries.EntryQuery) ([]domainentries.ListedEntry, error) {
	where := []string{"e.quantity > 0"}
	args := []any{}
	if query.OfficerID != nil {
		where = append(where, "e.officer_id = ?")
		args = append(args, *query.OfficerID)
	}
	if query.DistrictID != nil {
		where = append(where, "u.district_id = ?")
		args = append(args, *query.DistrictID)
	}
	if strings.TrimSpace(query.Month) != "" {
		where = append(where, "e.month = ?")
		args = append(args, strings.TrimSpace(query.Month))
	} else if len(query.Months) > 0 {
		where = append(where, "e.month IN ("+placeholders(len(query.Months))+")")
		for _, month := range query.Months {
			args = append(args, month)
		}
	} else if strings.TrimSpace(query.Year) != "" {
		where = append(where, "e.month LIKE ?")
		args = append(args, strings.TrimSpace(query.Year)+"-%")
	}
	if strings.TrimSpace(query.Status) != "" {
		where = append(where, "e.status = ?")
		args = append(args, strings.TrimSpace(query.Status))
	}

	sortDir := "DESC"
	if query.SortAscending {
		sortDir = "ASC"
	}
	sqlText := listedEntrySelect + " WHERE " + strings.Join(where, " AND ") +
		" ORDER BY " + entrySortColumn(query.SortKey) + " " + sortDir + ", e.id " + sortDir
	if query.Offset != nil {
		if query.Limit != nil {
			sqlText += " LIMIT ? OFFSET ?"
			args = append(args, *query.Limit, *query.Offset)
		} else {
			sqlText += " OFFSET ?"
			args = append(args, *query.Offset)
		}
	} else if query.Limit != nil {
		sqlText += " LIMIT ?"
		args = append(args, *query.Limit)
	}

	rows, err := r.q.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []domainentries.ListedEntry{}
	for rows.Next() {
		entry, err := scanListedEntry(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, entry)
	}
	return result, rows.Err()
}

func (r Repository) GetEntry(ctx context.Context, id int) (domainentries.ListedEntry, error) {
	entry, err := scanListedEntry(r.q.QueryRowContext(ctx, listedEntrySelect+` WHERE e.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domainentries.ListedEntry{}, errors.Join(sql.ErrNoRows, domainentries.ErrEntryNotFound)
	}
	return entry, err
}

func scanListedEntry(scanner rowScanner) (domainentries.ListedEntry, error) {
	var entry domainentries.ListedEntry
	var officerName sql.NullString
	var rank sql.NullString
	var districtName sql.NullString
	var code sql.NullString
	var indicator sql.NullString
	var block sql.NullString
	var subblock sql.NullString
	var npa sql.NullString
	var sourceURL sql.NullString
	var monthlyLimit sql.NullFloat64
	var isMass sql.NullInt64
	var isCritical sql.NullInt64
	if err := scanner.Scan(
		&entry.ID,
		&entry.Date,
		&entry.Month,
		&entry.OfficerID,
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
		&entry.Comments,
		&entry.HeadComment,
		&entry.AttachmentName,
		&entry.AttachmentStorageName,
		&entry.AttachmentContentType,
		&entry.AttachmentSize,
		&officerName,
		&rank,
		&districtName,
		&code,
		&indicator,
		&block,
		&subblock,
		&npa,
		&sourceURL,
		&monthlyLimit,
		&isMass,
		&isCritical,
	); err != nil {
		return domainentries.ListedEntry{}, err
	}
	entry.OfficerName = officerName.String
	entry.Rank = rank.String
	entry.DistrictName = districtName.String
	entry.Code = code.String
	entry.Indicator = indicator.String
	entry.Block = block.String
	entry.Subblock = subblock.String
	entry.NPA = npa.String
	entry.SourceURL = sourceURL.String
	entry.MonthlyLimit = monthlyLimit.Float64
	entry.IsMass = isMass.Valid && isMass.Int64 != 0
	entry.IsCritical = isCritical.Valid && isCritical.Int64 != 0
	return entry, nil
}

func entrySortColumn(sortKey string) string {
	switch sortKey {
	case "month":
		return "e.month"
	case "final_score":
		return "e.final_score"
	case "status":
		return "e.status"
	case "officer_name":
		return "u.full_name"
	case "code":
		return "c.code"
	default:
		return "e.date"
	}
}

func placeholders(count int) string {
	if count <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", count), ",")
}
