package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	dialectPostgres = "postgres"
	dialectSQLite   = "sqlite"
)

type Store struct {
	db      *sql.DB
	dialect string
}

type Tx struct {
	tx      *sql.Tx
	dialect string
}

type dbtx interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (s *Store) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return s.db.ExecContext(ctx, rebind(query, s.dialect), args...)
}

func (s *Store) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return s.db.QueryContext(ctx, rebind(query, s.dialect), args...)
}

func (s *Store) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return s.db.QueryRowContext(ctx, rebind(query, s.dialect), args...)
}

func (s *Store) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := s.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &Tx{tx: tx, dialect: s.dialect}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (t *Tx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return t.tx.ExecContext(ctx, rebind(query, t.dialect), args...)
}

func (t *Tx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return t.tx.QueryContext(ctx, rebind(query, t.dialect), args...)
}

func (t *Tx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return t.tx.QueryRowContext(ctx, rebind(query, t.dialect), args...)
}

func (t *Tx) Commit() error {
	return t.tx.Commit()
}

func (t *Tx) Rollback() error {
	return t.tx.Rollback()
}

func rebind(query string, dialect string) string {
	if dialect != dialectPostgres {
		return query
	}
	var builder strings.Builder
	builder.Grow(len(query) + 8)
	inSingleQuote := false
	index := 1
	for i := 0; i < len(query); i++ {
		ch := query[i]
		if ch == '\'' {
			builder.WriteByte(ch)
			if inSingleQuote && i+1 < len(query) && query[i+1] == '\'' {
				i++
				builder.WriteByte(query[i])
				continue
			}
			inSingleQuote = !inSingleQuote
			continue
		}
		if ch == '?' && !inSingleQuote {
			builder.WriteString("$")
			builder.WriteString(strconv.Itoa(index))
			index++
			continue
		}
		builder.WriteByte(ch)
	}
	return builder.String()
}

func createSchema(db *Store) error {
	idType := "INTEGER PRIMARY KEY AUTOINCREMENT"
	if db.dialect == dialectPostgres {
		idType = "SERIAL PRIMARY KEY"
	}
	statements := []string{
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS districts (
			id %s,
			name TEXT UNIQUE NOT NULL,
			coeff REAL NOT NULL DEFAULT 1.0
		)`, idType),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS users (
			id %s,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			full_name TEXT NOT NULL,
			rank TEXT NOT NULL DEFAULT '',
			role TEXT NOT NULL DEFAULT 'officer',
			district_id INTEGER REFERENCES districts(id),
			source_employee_number TEXT UNIQUE,
			monthly_norm REAL NOT NULL DEFAULT 100.0
		)`, idType),
		`CREATE INDEX IF NOT EXISTS ix_users_username ON users(username)`,
		`CREATE INDEX IF NOT EXISTS ix_users_role_district ON users(role, district_id)`,
		`CREATE INDEX IF NOT EXISTS ix_users_source_employee_number ON users(source_employee_number)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS ux_users_source_employee_number ON users(source_employee_number)`,
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS kpi_catalog (
			id %s,
			code TEXT UNIQUE NOT NULL,
			indicator TEXT NOT NULL,
			block TEXT NOT NULL,
			subblock TEXT NOT NULL DEFAULT '',
			npa TEXT NOT NULL DEFAULT '',
			points_per_unit REAL NOT NULL,
			unit TEXT NOT NULL DEFAULT 'ед.',
			source_url TEXT NOT NULL DEFAULT '',
			monthly_limit REAL NOT NULL DEFAULT 0,
			is_mass INTEGER NOT NULL DEFAULT 0,
			is_critical INTEGER NOT NULL DEFAULT 0,
			is_active INTEGER NOT NULL DEFAULT 1
		)`, idType),
		`CREATE INDEX IF NOT EXISTS ix_kpi_catalog_active_positive ON kpi_catalog(is_active, points_per_unit, block, code)`,
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS import_batches (
			id %s,
			source_name TEXT NOT NULL,
			source_sheet TEXT NOT NULL,
			source_checksum TEXT UNIQUE NOT NULL,
			month TEXT NOT NULL,
			employee_count INTEGER NOT NULL,
			kpi_count INTEGER NOT NULL,
			entry_count INTEGER NOT NULL,
			imported_at TEXT NOT NULL
		)`, idType),
		`CREATE INDEX IF NOT EXISTS ix_import_batches_checksum ON import_batches(source_checksum)`,
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS kpi_entries (
			id %s,
			date TEXT NOT NULL,
			month TEXT NOT NULL,
			officer_id INTEGER NOT NULL REFERENCES users(id),
			catalog_id INTEGER NOT NULL REFERENCES kpi_catalog(id),
			quantity REAL NOT NULL DEFAULT 1,
			counted_quantity REAL NOT NULL DEFAULT 0,
			points_per_unit REAL NOT NULL,
			coeff REAL NOT NULL DEFAULT 1.0,
			raw_score REAL NOT NULL DEFAULT 0,
			capped_score REAL NOT NULL DEFAULT 0,
			anti_coeff REAL NOT NULL DEFAULT 1.0,
			final_score REAL NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'pending',
			comments TEXT NOT NULL DEFAULT '',
			head_comment TEXT NOT NULL DEFAULT '',
			attachment_name TEXT NOT NULL DEFAULT '',
			attachment_storage_name TEXT NOT NULL DEFAULT '',
			attachment_content_type TEXT NOT NULL DEFAULT '',
			attachment_size INTEGER NOT NULL DEFAULT 0,
			source_key TEXT UNIQUE,
			source_row INTEGER,
			is_monthly_aggregate INTEGER NOT NULL DEFAULT 0,
			import_batch_id INTEGER REFERENCES import_batches(id)
		)`, idType),
		`CREATE INDEX IF NOT EXISTS ix_kpi_entries_officer_month ON kpi_entries(officer_id, month)`,
		`CREATE INDEX IF NOT EXISTS ix_kpi_entries_month_status ON kpi_entries(month, status)`,
		`CREATE INDEX IF NOT EXISTS ix_kpi_entries_month_officer_status ON kpi_entries(month, officer_id, status)`,
		`CREATE INDEX IF NOT EXISTS ix_kpi_entries_source_key ON kpi_entries(source_key)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS ux_kpi_entries_source_key ON kpi_entries(source_key)`,
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS entry_attachments (
			id %s,
			entry_id INTEGER NOT NULL REFERENCES kpi_entries(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			storage_name TEXT NOT NULL,
			content_type TEXT NOT NULL,
			size INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			UNIQUE(entry_id, storage_name)
		)`, idType),
		`CREATE INDEX IF NOT EXISTS ix_entry_attachments_entry_id ON entry_attachments(entry_id, id)`,
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS kpi_targets (
			id %s,
			month TEXT NOT NULL,
			officer_id INTEGER NOT NULL REFERENCES users(id),
			catalog_id INTEGER NOT NULL REFERENCES kpi_catalog(id),
			target_quantity REAL NOT NULL DEFAULT 0,
			UNIQUE(month, officer_id, catalog_id)
		)`, idType),
		`CREATE INDEX IF NOT EXISTS ix_kpi_targets_month_officer ON kpi_targets(month, officer_id)`,
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS app_notifications (
			id %s,
			user_id INTEGER NOT NULL REFERENCES users(id),
			kind TEXT NOT NULL,
			title TEXT NOT NULL,
			message TEXT NOT NULL,
			link TEXT NOT NULL DEFAULT '',
			is_read INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		)`, idType),
		`CREATE INDEX IF NOT EXISTS ix_app_notifications_user_read ON app_notifications(user_id, is_read, created_at)`,
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS direct_messages (
			id %s,
			sender_id INTEGER NOT NULL REFERENCES users(id),
			recipient_id INTEGER NOT NULL REFERENCES users(id),
			entry_id INTEGER REFERENCES kpi_entries(id),
			body TEXT NOT NULL,
			created_at TEXT NOT NULL,
			read_at TEXT,
			deleted_at TEXT
		)`, idType),
		`CREATE INDEX IF NOT EXISTS ix_direct_messages_sender_recipient ON direct_messages(sender_id, recipient_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS ix_direct_messages_recipient_read ON direct_messages(recipient_id, read_at, deleted_at)`,
		`CREATE INDEX IF NOT EXISTS ix_direct_messages_recipient_unread ON direct_messages(recipient_id, read_at, deleted_at, id)`,
		`CREATE INDEX IF NOT EXISTS ix_direct_messages_conversation_id ON direct_messages(sender_id, recipient_id, id)`,
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS support_conversations (
			id %s,
			user_id INTEGER NOT NULL UNIQUE REFERENCES users(id),
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`, idType),
		`CREATE INDEX IF NOT EXISTS ix_support_conversations_user_id ON support_conversations(user_id)`,
		`CREATE INDEX IF NOT EXISTS ix_support_conversations_updated_at ON support_conversations(updated_at)`,
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS support_messages (
			id %s,
			conversation_id INTEGER NOT NULL REFERENCES support_conversations(id),
			sender_id INTEGER NOT NULL REFERENCES users(id),
			body TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`, idType),
		`CREATE INDEX IF NOT EXISTS ix_support_messages_conversation_id ON support_messages(conversation_id, created_at, id)`,
		`CREATE INDEX IF NOT EXISTS ix_support_messages_sender_id ON support_messages(sender_id)`,
	}
	for _, stmt := range statements {
		if _, err := db.ExecContext(context.Background(), stmt); err != nil {
			return err
		}
	}
	if err := ensureColumns(db); err != nil {
		return err
	}
	if err := migrateLeadershipScope(db); err != nil {
		return err
	}
	_, err := db.ExecContext(context.Background(), `INSERT INTO entry_attachments
		(entry_id, name, storage_name, content_type, size, created_at)
		SELECT id, attachment_name, attachment_storage_name, attachment_content_type, attachment_size, ?
		FROM kpi_entries
		WHERE attachment_storage_name <> ''
		ON CONFLICT(entry_id, storage_name) DO NOTHING`, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func ensureColumns(db *Store) error {
	columns := map[string][]string{
		"users": {
			`source_employee_number TEXT`,
			`monthly_norm REAL NOT NULL DEFAULT 100.0`,
			`leadership_scope TEXT NOT NULL DEFAULT ''`,
		},
		"kpi_catalog": {
			`subblock TEXT NOT NULL DEFAULT ''`,
			`npa TEXT NOT NULL DEFAULT ''`,
			`source_url TEXT NOT NULL DEFAULT ''`,
			`monthly_limit REAL NOT NULL DEFAULT 0`,
			`is_mass INTEGER NOT NULL DEFAULT 0`,
			`is_critical INTEGER NOT NULL DEFAULT 0`,
			`is_active INTEGER NOT NULL DEFAULT 1`,
		},
		"kpi_entries": {
			`counted_quantity REAL NOT NULL DEFAULT 0`,
			`capped_score REAL NOT NULL DEFAULT 0`,
			`anti_coeff REAL NOT NULL DEFAULT 1.0`,
			`attachment_name TEXT NOT NULL DEFAULT ''`,
			`attachment_storage_name TEXT NOT NULL DEFAULT ''`,
			`attachment_content_type TEXT NOT NULL DEFAULT ''`,
			`attachment_size INTEGER NOT NULL DEFAULT 0`,
			`source_key TEXT`,
			`source_row INTEGER`,
			`is_monthly_aggregate INTEGER NOT NULL DEFAULT 0`,
			`import_batch_id INTEGER REFERENCES import_batches(id)`,
		},
		"direct_messages": {
			`entry_id INTEGER REFERENCES kpi_entries(id)`,
			`deleted_at TEXT`,
		},
	}
	for table, definitions := range columns {
		existing, err := tableColumns(db, table)
		if err != nil {
			return err
		}
		for _, definition := range definitions {
			name := strings.Fields(definition)[0]
			if existing[name] {
				continue
			}
			if _, err := db.ExecContext(context.Background(), fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", table, definition)); err != nil {
				return err
			}
		}
	}
	return nil
}

func tableColumns(db *Store, table string) (map[string]bool, error) {
	if db.dialect == dialectPostgres {
		rows, err := db.QueryContext(context.Background(), `SELECT column_name
			FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = ?`, table)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		columns := map[string]bool{}
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				return nil, err
			}
			columns[name] = true
		}
		return columns, rows.Err()
	}
	rows, err := db.QueryContext(context.Background(), fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			return nil, err
		}
		columns[name] = true
	}
	return columns, rows.Err()
}

func parseInt(value string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

func parseOptionalInt(value string) (*int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := parseInt(value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func parseFloat(value string, fallback float64) (float64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}
	return strconv.ParseFloat(value, 64)
}

func parseBoolInt(value string) bool {
	value = strings.TrimSpace(value)
	return value == "1" || strings.EqualFold(value, "true")
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func round1(value float64) float64 {
	return math.Round(value*10) / 10
}

func round6(value float64) float64 {
	return math.Round(value*1_000_000) / 1_000_000
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func monthFromDate(date string) string {
	if len(date) >= 7 {
		return date[:7]
	}
	return ""
}

func todayDate() string {
	return time.Now().Format("2006-01-02")
}

func currentMonth() string {
	return time.Now().Format("2006-01")
}

func nowDB() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
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

func mustDate(value string) bool {
	if len(value) != 10 {
		return false
	}
	_, err := time.Parse("2006-01-02", value)
	return err == nil
}

func validateMonth(month string) error {
	if len(month) != 7 || month[4] != '-' {
		return errors.New("Месяц указан неверно")
	}
	year, err1 := strconv.Atoi(month[:4])
	monthNum, err2 := strconv.Atoi(month[5:])
	if err1 != nil || err2 != nil || year < 1 || monthNum < 1 || monthNum > 12 {
		return errors.New("Месяц указан неверно")
	}
	return nil
}

func monthShift(base time.Time, offset int) string {
	year, month, _ := base.Date()
	total := year*12 + int(month) - 1 + offset
	newYear := total / 12
	newMonth := total%12 + 1
	return fmt.Sprintf("%04d-%02d", newYear, newMonth)
}
