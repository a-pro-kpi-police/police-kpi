package entries

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	domainentries "kpi-police/backend-golang/internal/domain/entries"
)

type Queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Tx interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	Commit() error
	Rollback() error
}

type BeginTxFunc func(context.Context, *sql.TxOptions) (Tx, error)

type RecalculateFunc func(context.Context, Tx, int, string) error

type Repository struct {
	q           Queryer
	beginTx     BeginTxFunc
	recalculate RecalculateFunc
}

func NewRepository(q Queryer, beginTx BeginTxFunc, recalculate RecalculateFunc) Repository {
	return Repository{q: q, beginTx: beginTx, recalculate: recalculate}
}

func (r Repository) GetOfficer(ctx context.Context, id int) (domainentries.Officer, error) {
	var officer domainentries.Officer
	err := r.q.QueryRowContext(ctx, `SELECT u.id, COALESCE(d.coeff, 1.0)
		FROM users u
		LEFT JOIN districts d ON d.id = u.district_id
		WHERE u.id = ? AND u.role = ?`,
		id, domainentries.RoleOfficer,
	).Scan(&officer.ID, &officer.Coeff)
	if err == sql.ErrNoRows {
		return domainentries.Officer{}, domainentries.ErrOfficerNotFound
	}
	if err != nil {
		return domainentries.Officer{}, err
	}
	return officer, nil
}

func (r Repository) GetCatalog(ctx context.Context, id int) (domainentries.Catalog, error) {
	var catalog domainentries.Catalog
	var isActive int
	err := r.q.QueryRowContext(ctx, `SELECT id, code, points_per_unit, monthly_limit, is_active
		FROM kpi_catalog
		WHERE id = ?`,
		id,
	).Scan(&catalog.ID, &catalog.Code, &catalog.PointsPerUnit, &catalog.MonthlyLimit, &isActive)
	if err == sql.ErrNoRows {
		return domainentries.Catalog{}, domainentries.ErrCatalogNotFound
	}
	if err != nil {
		return domainentries.Catalog{}, err
	}
	catalog.IsActive = isActive != 0
	return catalog, nil
}

func (r Repository) CreateEntry(ctx context.Context, actor domainentries.Actor, entry domainentries.Entry) (domainentries.Entry, error) {
	tx, err := r.beginTx(ctx, nil)
	if err != nil {
		return domainentries.Entry{}, err
	}
	defer tx.Rollback()

	var entryID int
	err = tx.QueryRowContext(ctx, `INSERT INTO kpi_entries
		(date, month, officer_id, catalog_id, quantity, counted_quantity, points_per_unit, coeff,
		 raw_score, capped_score, anti_coeff, final_score, status, comments, head_comment,
		 attachment_name, attachment_storage_name, attachment_content_type, attachment_size)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id`,
		entry.Date,
		entry.Month,
		entry.OfficerID,
		entry.CatalogID,
		entry.Quantity,
		entry.CountedQuantity,
		entry.PointsPerUnit,
		entry.Coeff,
		entry.RawScore,
		entry.CappedScore,
		entry.AntiCoeff,
		entry.FinalScore,
		entry.Status,
		entry.Comments,
		entry.HeadComment,
		entry.AttachmentName,
		entry.AttachmentStorageName,
		entry.AttachmentContentType,
		entry.AttachmentSize,
	).Scan(&entryID)
	if err != nil {
		return domainentries.Entry{}, err
	}
	entry.ID = entryID

	if actor.Role == domainentries.RoleOfficer {
		code := "KPI"
		if strings.TrimSpace(entry.CatalogCode) != "" {
			code = entry.CatalogCode
		}
		if err := notifyHeads(ctx, tx, "entry_pending", "Новая запись KPI", fmt.Sprintf("%s отправил запись %s на подтверждение.", actor.FullName, code), "/entries"); err != nil {
			return domainentries.Entry{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return domainentries.Entry{}, err
	}
	return entry, nil
}

func (r Repository) GetEntryForStatusUpdate(ctx context.Context, id int) (domainentries.Entry, error) {
	var entry domainentries.Entry
	var catalogCode sql.NullString
	err := r.q.QueryRowContext(ctx, `SELECT
			e.id, e.date, e.month, e.officer_id, e.catalog_id, e.quantity,
			e.status, e.head_comment, c.code
		FROM kpi_entries e
		LEFT JOIN kpi_catalog c ON c.id = e.catalog_id
		WHERE e.id = ?`,
		id,
	).Scan(
		&entry.ID,
		&entry.Date,
		&entry.Month,
		&entry.OfficerID,
		&entry.CatalogID,
		&entry.Quantity,
		&entry.Status,
		&entry.HeadComment,
		&catalogCode,
	)
	if err == sql.ErrNoRows {
		return domainentries.Entry{}, domainentries.ErrEntryNotFound
	}
	if err != nil {
		return domainentries.Entry{}, err
	}
	if catalogCode.Valid {
		entry.CatalogCode = catalogCode.String
	}
	return entry, nil
}

func (r Repository) ApplyStatusChange(ctx context.Context, change domainentries.StatusChange) (domainentries.Entry, error) {
	tx, err := r.beginTx(ctx, nil)
	if err != nil {
		return domainentries.Entry{}, err
	}
	defer tx.Rollback()

	entry := change.Entry
	if _, err := tx.ExecContext(ctx, `UPDATE kpi_entries SET status = ?, head_comment = ? WHERE id = ?`,
		entry.Status, entry.HeadComment, entry.ID); err != nil {
		return domainentries.Entry{}, err
	}
	if r.recalculate != nil {
		if err := r.recalculate(ctx, tx, entry.OfficerID, entry.Month); err != nil {
			return domainentries.Entry{}, err
		}
	}
	if change.NotifyOfficer {
		title, message := statusNotification(entry.Status, entry.CatalogCode)
		if err := addNotification(ctx, tx, entry.OfficerID, "entry_"+entry.Status, title, message, "/entries"); err != nil {
			return domainentries.Entry{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return domainentries.Entry{}, err
	}
	return entry, nil
}

func notifyHeads(ctx context.Context, q Tx, kind string, title string, message string, link string) error {
	rows, err := q.QueryContext(ctx, `SELECT id FROM users WHERE role = ?`, domainentries.RoleHead)
	if err != nil {
		return err
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range ids {
		if err := addNotification(ctx, q, id, kind, title, message, link); err != nil {
			return err
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

func statusNotification(status string, code string) (string, string) {
	if strings.TrimSpace(code) == "" {
		code = "KPI"
	}
	if status == domainentries.StatusConfirmed {
		return "Запись KPI подтверждена", fmt.Sprintf("Запись %s подтверждена начальником.", code)
	}
	return "Запись KPI отклонена", fmt.Sprintf("Запись %s отклонена начальником.", code)
}
