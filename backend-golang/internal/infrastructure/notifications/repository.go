package notifications

import (
	"context"
	"database/sql"
	"time"

	domainnotifications "kpi-police/backend-golang/internal/domain/notifications"
)

type Queryer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Repository struct {
	q Queryer
}

func NewRepository(q Queryer) Repository {
	return Repository{q: q}
}

func (r Repository) List(ctx context.Context, userID int, limit int) (domainnotifications.List, error) {
	var result domainnotifications.List
	if err := r.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM app_notifications WHERE user_id = ? AND is_read = 0`, userID).
		Scan(&result.UnreadCount); err != nil {
		return domainnotifications.List{}, err
	}

	rows, err := r.q.QueryContext(ctx, notificationSelect+`
		WHERE user_id = ?
		ORDER BY created_at DESC, id DESC
		LIMIT ?`, userID, limit)
	if err != nil {
		return domainnotifications.List{}, err
	}
	defer rows.Close()

	for rows.Next() {
		item, err := scanNotification(rows)
		if err != nil {
			return domainnotifications.List{}, err
		}
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		return domainnotifications.List{}, err
	}
	return result, nil
}

func (r Repository) MarkRead(ctx context.Context, userID int, notificationID int) (domainnotifications.Notification, error) {
	result, err := r.q.ExecContext(ctx, `UPDATE app_notifications SET is_read = 1 WHERE id = ? AND user_id = ?`, notificationID, userID)
	if err != nil {
		return domainnotifications.Notification{}, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return domainnotifications.Notification{}, domainnotifications.ErrNotFound
	}
	return r.queryNotification(ctx, notificationID, userID)
}

func (r Repository) MarkAllRead(ctx context.Context, userID int) (int64, error) {
	result, err := r.q.ExecContext(ctx, `UPDATE app_notifications SET is_read = 1 WHERE user_id = ? AND is_read = 0`, userID)
	if err != nil {
		return 0, err
	}
	updated, _ := result.RowsAffected()
	return updated, nil
}

func (r Repository) queryNotification(ctx context.Context, id int, userID int) (domainnotifications.Notification, error) {
	item, err := scanNotification(r.q.QueryRowContext(ctx, notificationSelect+` WHERE id = ? AND user_id = ?`, id, userID))
	if err == sql.ErrNoRows {
		return domainnotifications.Notification{}, domainnotifications.ErrNotFound
	}
	return item, err
}

func scanNotification(scanner interface {
	Scan(dest ...any) error
}) (domainnotifications.Notification, error) {
	var item domainnotifications.Notification
	var isRead int
	var createdAt sql.NullString
	if err := scanner.Scan(&item.ID, &item.Kind, &item.Title, &item.Message, &item.Link, &isRead, &createdAt); err != nil {
		return domainnotifications.Notification{}, err
	}
	item.IsRead = isRead != 0
	if parsed := parseDBTime(createdAt); parsed != nil {
		item.CreatedAt = *parsed
	} else {
		item.CreatedAt = time.Now().UTC()
	}
	return item, nil
}

func parseDBTime(value sql.NullString) *time.Time {
	if !value.Valid || value.String == "" {
		return nil
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999-07:00",
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value.String); err == nil {
			utc := parsed.UTC()
			return &utc
		}
	}
	return nil
}

const notificationSelect = `SELECT id, kind, title, message, link, is_read, created_at
FROM app_notifications`
