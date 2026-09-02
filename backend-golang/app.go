package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

type App struct {
	db        *Store
	secret    []byte
	uploadDir string
}

func NewApp() (*App, error) {
	driver, dsn, dialect := databaseConfig(os.Getenv("KPI_DATABASE_URL"))
	logDebugf("app init: database driver=%s dialect=%s debug=%t", driver, dialect, debugLoggingEnabled())
	db, err := sql.Open(driver, dsn)
	if err != nil {
		logErrorf("app init: open database failed: %v", err)
		return nil, err
	}
	if dialect == dialectSQLite {
		db.SetMaxOpenConns(1)
		if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
			logErrorf("app init: enable sqlite foreign keys failed: %v", err)
			return nil, err
		}
		logDebugf("app init: sqlite foreign keys enabled")
	} else {
		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(5)
		logDebugf("app init: postgres pool configured max_open=%d max_idle=%d", 10, 5)
	}
	store := &Store{db: db, dialect: dialect}
	if err := createSchema(store); err != nil {
		logErrorf("app init: schema migration failed: %v", err)
		return nil, err
	}
	logDebugf("app init: schema ready")
	uploadDir := os.Getenv("KPI_UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}
	logDebugf("app init: upload_dir=%s", uploadDir)
	app := &App{
		db:        store,
		secret:    []byte(envDefault("KPI_SECRET", "change-me-in-production-please")),
		uploadDir: uploadDir,
	}
	if err := app.seed(); err != nil {
		logErrorf("app init: seed failed: %v", err)
		return nil, err
	}
	logDebugf("app init: seed complete")
	return app, nil
}

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/auth/login", a.handleLogin)
	mux.HandleFunc("GET /api/auth/me", a.handleMe)

	mux.HandleFunc("GET /api/catalog", a.handleCatalog)
	mux.HandleFunc("GET /api/districts", a.handleDistricts)
	mux.HandleFunc("GET /api/officers", a.handleOfficers)
	mux.HandleFunc("GET /api/rating", a.handleRating)
	mux.HandleFunc("GET /api/months", a.handleMonths)

	mux.HandleFunc("GET /api/admin/users", a.handleAdminListUsers)
	mux.HandleFunc("POST /api/admin/users", a.handleAdminCreateUser)
	mux.HandleFunc("PATCH /api/admin/users/{user_id}", a.handleAdminUpdateUser)
	mux.HandleFunc("DELETE /api/admin/users/{user_id}", a.handleAdminDeleteUser)
	mux.HandleFunc("GET /api/admin/rating", a.handleAdminRating)
	mux.HandleFunc("GET /api/admin/catalog", a.handleAdminListCatalog)
	mux.HandleFunc("POST /api/admin/catalog", a.handleAdminCreateCatalogItem)
	mux.HandleFunc("PATCH /api/admin/catalog/{catalog_id}", a.handleAdminUpdateCatalogItem)
	mux.HandleFunc("DELETE /api/admin/catalog/{catalog_id}", a.handleAdminDeleteCatalogItem)

	mux.HandleFunc("GET /api/entries", a.handleListEntries)
	mux.HandleFunc("POST /api/entries", a.handleCreateEntry)
	mux.HandleFunc("POST /api/entries/with-attachment", a.handleCreateEntryWithAttachment)
	mux.HandleFunc("GET /api/entries/{entry_id}/attachment", a.handleDownloadAttachment)
	mux.HandleFunc("PATCH /api/entries/{entry_id}/status", a.handleUpdateEntryStatus)
	mux.HandleFunc("DELETE /api/entries/{entry_id}", a.handleDeleteEntry)

	mux.HandleFunc("GET /api/dashboard", a.handleDashboard)

	mux.HandleFunc("GET /api/targets", a.handleGetTargets)
	mux.HandleFunc("PUT /api/targets", a.handleSaveTargets)
	mux.HandleFunc("GET /api/group-targets", a.handleGetGroupTargets)
	mux.HandleFunc("PUT /api/group-targets", a.handleSaveGroupTargets)
	mux.HandleFunc("GET /api/target-progress", a.handleTargetProgress)
	mux.HandleFunc("GET /api/target-overview", a.handleTargetOverview)

	mux.HandleFunc("GET /api/notifications", a.handleListNotifications)
	mux.HandleFunc("PATCH /api/notifications/{notification_id}/read", a.handleMarkNotificationRead)
	mux.HandleFunc("POST /api/notifications/read-all", a.handleMarkAllNotificationsRead)

	mux.HandleFunc("GET /api/messenger/contacts", a.handleMessengerContacts)
	mux.HandleFunc("GET /api/messenger/unread-count", a.handleMessengerUnreadCount)
	mux.HandleFunc("GET /api/messenger/entries/{contact_id}", a.handleMessengerEntries)
	mux.HandleFunc("GET /api/messenger/messages/{contact_id}", a.handleMessengerMessages)
	mux.HandleFunc("POST /api/messenger/messages", a.handleSendMessage)
	mux.HandleFunc("DELETE /api/messenger/messages/{message_id}", a.handleDeleteMessage)

	mux.HandleFunc("GET /api/support/conversation", a.handleSupportConversation)
	mux.HandleFunc("POST /api/support/messages", a.handleSupportSendMessage)
	mux.HandleFunc("GET /api/admin/support/conversations", a.handleAdminSupportConversations)
	mux.HandleFunc("GET /api/admin/support/conversations/{conversation_id}/messages", a.handleAdminSupportMessages)
	mux.HandleFunc("POST /api/admin/support/conversations/{conversation_id}/messages", a.handleAdminSupportSendMessage)

	return corsMiddleware(loggingMiddleware(recoveryMiddleware(mux)))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := newStatusRecorder(w)
		next.ServeHTTP(recorder, r)
		duration := time.Since(start).Round(time.Millisecond)
		if recorder.status >= http.StatusInternalServerError {
			logErrorf("request method=%s path=%s status=%d bytes=%d duration=%s", r.Method, r.URL.Path, recorder.status, recorder.bytes, duration)
			return
		}
		logInfof("request method=%s path=%s status=%d bytes=%d duration=%s", r.Method, r.URL.Path, recorder.status, recorder.bytes, duration)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		logErrorf("write json failed: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, detail string) {
	if status >= http.StatusInternalServerError {
		logErrorf("http error status=%d detail=%q", status, detail)
	} else if status >= http.StatusBadRequest {
		logDebugf("http client error status=%d detail=%q", status, detail)
	}
	writeJSON(w, status, map[string]string{"detail": detail})
}

func readJSON(r *http.Request, dest any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dest)
}

func envDefault(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func databaseConfig(raw string) (driver string, dsn string, dialect string) {
	if raw == "" {
		raw = databaseURLFromPostgresEnv()
	}
	if raw == "" {
		raw = "postgres://kpi:kpi@localhost:5432/kpi?sslmode=disable"
	}
	if strings.HasPrefix(raw, "postgres://") || strings.HasPrefix(raw, "postgresql://") || strings.HasPrefix(raw, "host=") {
		return "postgres", raw, dialectPostgres
	}
	if raw == "sqlite:///:memory:" {
		return "sqlite", ":memory:", dialectSQLite
	}
	const prefix = "sqlite:///"
	if strings.HasPrefix(raw, prefix) {
		path := strings.TrimPrefix(raw, prefix)
		if strings.HasPrefix(path, "/") {
			return "sqlite", path, dialectSQLite
		}
		return "sqlite", path, dialectSQLite
	}
	return "postgres", raw, dialectPostgres
}

func databaseURLFromPostgresEnv() string {
	password := os.Getenv("POSTGRES_PASSWORD")
	if password == "" {
		return ""
	}
	user := envDefault("POSTGRES_USER", "kpi")
	dbName := envDefault("POSTGRES_DB", "kpi")
	host := envDefault("POSTGRES_HOST", "postgres")
	port := envDefault("POSTGRES_PORT", "5432")
	value := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(user, password),
		Host:     net.JoinHostPort(host, port),
		Path:     dbName,
		RawQuery: "sslmode=disable",
	}
	return value.String()
}

func parseIDPathValue(r *http.Request, name string) (int, bool) {
	id, err := parseInt(r.PathValue(name))
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func requireUser(a *App, w http.ResponseWriter, r *http.Request) (*User, bool) {
	user, err := a.currentUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Не удалось проверить учётные данные")
		return nil, false
	}
	return user, true
}

func requireHead(a *App, w http.ResponseWriter, r *http.Request) (*User, bool) {
	user, ok := requireUser(a, w, r)
	if !ok {
		return nil, false
	}
	if user.Role != roleHead {
		writeError(w, http.StatusForbidden, "Доступно только начальнику отдела")
		return nil, false
	}
	return user, true
}

func requireAdmin(a *App, w http.ResponseWriter, r *http.Request) (*User, bool) {
	user, ok := requireUser(a, w, r)
	if !ok {
		return nil, false
	}
	if user.Role != roleAdmin {
		writeError(w, http.StatusForbidden, "Доступно только администратору")
		return nil, false
	}
	return user, true
}

func requireHeadOrAdmin(a *App, w http.ResponseWriter, r *http.Request) (*User, bool) {
	user, ok := requireUser(a, w, r)
	if !ok {
		return nil, false
	}
	if !isEntryManagerRole(user.Role) {
		writeError(w, http.StatusForbidden, "Недостаточно прав")
		return nil, false
	}
	return user, true
}

func isEntryManagerRole(role string) bool {
	return role == roleHead || role == roleAdmin
}

func (a *App) currentUser(r *http.Request) (*User, error) {
	header := r.Header.Get("Authorization")
	token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if token == "" || token == header {
		return nil, errors.New("missing bearer token")
	}
	claims, err := a.parseToken(token)
	if err != nil {
		return nil, err
	}
	return a.getUserByID(r.Context(), claims.UserID)
}
