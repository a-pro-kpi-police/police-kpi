package messenger

import (
	"context"
	"database/sql"
	"strings"
	"time"

	domainmessenger "kpi-police/backend-golang/internal/domain/messenger"
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

func (r Repository) LastMessage(ctx context.Context, userID int, contactID int) (domainmessenger.Message, error) {
	message, err := scanMessage(r.q.QueryRowContext(ctx, messageSelect+conversationWhereSQL()+` ORDER BY m.created_at DESC, m.id DESC LIMIT 1`,
		userID, contactID, contactID, userID,
	))
	if err == sql.ErrNoRows {
		return domainmessenger.Message{}, domainmessenger.ErrMessageNotFound
	}
	return message, err
}

func (r Repository) CountUnread(ctx context.Context, userID int, contactID int) (int, error) {
	var count int
	err := r.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM direct_messages
		WHERE sender_id = ? AND recipient_id = ? AND read_at IS NULL AND deleted_at IS NULL`,
		contactID, userID,
	).Scan(&count)
	return count, err
}

func (r Repository) ListEntries(ctx context.Context, officerID int, limit int) ([]domainmessenger.Entry, error) {
	rows, err := r.q.QueryContext(ctx, messengerEntrySelect+` WHERE e.officer_id = ? AND e.quantity > 0 ORDER BY e.date DESC, e.id DESC LIMIT ?`,
		officerID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []domainmessenger.Entry
	for rows.Next() {
		entry, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (r Repository) EntryOfficerID(ctx context.Context, entryID int) (int, error) {
	var officerID int
	err := r.q.QueryRowContext(ctx, `SELECT officer_id FROM kpi_entries WHERE id = ?`, entryID).Scan(&officerID)
	if err == sql.ErrNoRows {
		return 0, domainmessenger.ErrEntryNotFound
	}
	return officerID, err
}

func (r Repository) ConversationMessages(ctx context.Context, userID int, contactID int) ([]domainmessenger.Message, error) {
	rows, err := r.q.QueryContext(ctx, messageSelect+conversationWhereSQL()+` ORDER BY m.created_at DESC, m.id DESC LIMIT 500`,
		userID, contactID, contactID, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []domainmessenger.Message
	for rows.Next() {
		message, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	if err := r.attachEntries(ctx, messages); err != nil {
		return nil, err
	}
	return messages, nil
}

func (r Repository) MarkMessagesRead(ctx context.Context, ids []int, readAt time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	args := []any{dbTime(readAt)}
	for _, id := range ids {
		args = append(args, id)
	}
	_, err := r.q.ExecContext(ctx, `UPDATE direct_messages SET read_at = ? WHERE id IN (`+placeholders(len(ids))+`)`, args...)
	return err
}

func (r Repository) CreateMessage(ctx context.Context, message domainmessenger.CreateMessage) (domainmessenger.Message, error) {
	var entryID any
	if message.EntryID != nil {
		entryID = *message.EntryID
	}
	var messageID int
	err := r.q.QueryRowContext(ctx, `INSERT INTO direct_messages
		(sender_id, recipient_id, entry_id, body, created_at)
		VALUES (?, ?, ?, ?, ?)
		RETURNING id`,
		message.SenderID, message.RecipientID, entryID, message.Body, dbTime(message.CreatedAt),
	).Scan(&messageID)
	if err != nil {
		return domainmessenger.Message{}, err
	}
	return r.GetMessageByID(ctx, messageID)
}

func (r Repository) GetMessageByID(ctx context.Context, id int) (domainmessenger.Message, error) {
	message, err := scanMessage(r.q.QueryRowContext(ctx, messageSelect+` WHERE m.id = ?`, id))
	if err == sql.ErrNoRows {
		return domainmessenger.Message{}, domainmessenger.ErrMessageNotFound
	}
	if err != nil {
		return domainmessenger.Message{}, err
	}
	if message.EntryID != nil {
		entry, err := r.entryByIDNoError(ctx, *message.EntryID)
		if err != nil {
			return domainmessenger.Message{}, err
		}
		message.Entry = entry
	}
	return message, nil
}

func (r Repository) DeleteMessage(ctx context.Context, id int, deletedAt time.Time) (domainmessenger.Message, error) {
	value := dbTime(deletedAt)
	if _, err := r.q.ExecContext(ctx, `UPDATE direct_messages
		SET deleted_at = ?, read_at = COALESCE(read_at, ?)
		WHERE id = ?`,
		value, value, id,
	); err != nil {
		return domainmessenger.Message{}, err
	}
	return r.GetMessageByID(ctx, id)
}

func (r Repository) attachEntries(ctx context.Context, messages []domainmessenger.Message) error {
	for i := range messages {
		if messages[i].EntryID == nil {
			continue
		}
		entry, err := r.entryByIDNoError(ctx, *messages[i].EntryID)
		if err != nil {
			return err
		}
		messages[i].Entry = entry
	}
	return nil
}

func (r Repository) entryByIDNoError(ctx context.Context, id int) (*domainmessenger.Entry, error) {
	entry, err := scanEntry(r.q.QueryRowContext(ctx, messengerEntrySelect+` WHERE e.id = ?`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func scanMessage(scanner interface {
	Scan(dest ...any) error
}) (domainmessenger.Message, error) {
	var message domainmessenger.Message
	var entryID sql.NullInt64
	var createdAt, readAt, deletedAt sql.NullString
	if err := scanner.Scan(&message.ID, &message.SenderID, &message.RecipientID, &entryID, &message.Body, &createdAt, &readAt, &deletedAt, &message.SenderName); err != nil {
		return domainmessenger.Message{}, err
	}
	if entryID.Valid {
		id := int(entryID.Int64)
		message.EntryID = &id
	}
	if parsed := parseDBTime(createdAt); parsed != nil {
		message.CreatedAt = *parsed
	} else {
		message.CreatedAt = time.Now().UTC()
	}
	message.ReadAt = parseDBTime(readAt)
	message.DeletedAt = parseDBTime(deletedAt)
	return message, nil
}

func scanEntry(scanner interface {
	Scan(dest ...any) error
}) (domainmessenger.Entry, error) {
	var entry domainmessenger.Entry
	err := scanner.Scan(
		&entry.ID,
		&entry.Date,
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
		&entry.Code,
		&entry.Indicator,
		&entry.Block,
		&entry.NPA,
		&entry.SourceURL,
		&entry.MonthlyLimit,
	)
	return entry, err
}

func parseDBTime(value sql.NullString) *time.Time {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
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

func dbTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func conversationWhereSQL() string {
	return ` WHERE ((m.sender_id = ? AND m.recipient_id = ?) OR (m.sender_id = ? AND m.recipient_id = ?))`
}

func placeholders(count int) string {
	if count <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", count), ",")
}

const messageSelect = `SELECT
	m.id, m.sender_id, m.recipient_id, m.entry_id, m.body, m.created_at, m.read_at, m.deleted_at,
	u.full_name
FROM direct_messages m
LEFT JOIN users u ON u.id = m.sender_id`

const messengerEntrySelect = `SELECT
	e.id, e.date, e.quantity, e.counted_quantity, e.points_per_unit, e.coeff,
	e.raw_score, e.capped_score, e.anti_coeff, e.final_score,
	e.status, e.comments, e.head_comment,
	c.code, c.indicator, c.block, c.npa, c.source_url, c.monthly_limit
FROM kpi_entries e
LEFT JOIN kpi_catalog c ON c.id = e.catalog_id`
