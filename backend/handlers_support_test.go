package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestSupportChatPersistsAndEnforcesPermissions(t *testing.T) {
	app := newSupportTestApp(t)
	defer app.db.Close()
	handler := app.routes()

	districtID := insertSupportDistrict(t, app)
	officer := insertSupportUser(t, app, "officer.support", "Иванов Иван", "Лейтенант полиции", roleOfficer, &districtID)
	otherOfficer := insertSupportUser(t, app, "other.support", "Петров Петр", "Сержант полиции", roleOfficer, &districtID)
	admin, err := app.getUserByUsername(context.Background(), "admin")
	if err != nil {
		t.Fatal(err)
	}

	officerToken := supportToken(t, app, officer)
	otherOfficerToken := supportToken(t, app, otherOfficer)
	adminToken := supportToken(t, app, admin)

	response := supportJSONRequest(t, handler, http.MethodGet, "/api/support/conversation", officerToken, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("initial user conversation status = %d, want %d", response.Code, http.StatusOK)
	}
	var emptyThread SupportThreadOut
	decodeSupportResponse(t, response, &emptyThread)
	if emptyThread.Conversation != nil || len(emptyThread.Messages) != 0 {
		t.Fatalf("initial user thread = %#v, want no conversation and no messages", emptyThread)
	}

	response = supportJSONRequest(t, handler, http.MethodPost, "/api/support/messages", adminToken, map[string]string{"body": "admin attempt"})
	if response.Code != http.StatusForbidden {
		t.Fatalf("admin user-send status = %d, want %d", response.Code, http.StatusForbidden)
	}

	response = supportJSONRequest(t, handler, http.MethodPost, "/api/support/messages", officerToken, map[string]string{"body": "   "})
	if response.Code != http.StatusBadRequest {
		t.Fatalf("empty first message status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	response = supportJSONRequest(t, handler, http.MethodGet, "/api/admin/support/conversations", adminToken, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("admin conversations after empty status = %d, want %d", response.Code, http.StatusOK)
	}
	var emptyConversations []SupportConversationOut
	decodeSupportResponse(t, response, &emptyConversations)
	if len(emptyConversations) != 0 {
		t.Fatalf("admin conversations after empty message length = %d, want 0", len(emptyConversations))
	}

	response = supportJSONRequest(t, handler, http.MethodPost, "/api/support/messages", officerToken, map[string]string{"body": "Нужна помощь"})
	if response.Code != http.StatusOK {
		t.Fatalf("user send status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	var userMessage SupportMessageOut
	decodeSupportResponse(t, response, &userMessage)
	if userMessage.Body != "Нужна помощь" || !userMessage.SentByMe || userMessage.SentByAdmin {
		t.Fatalf("user message = %#v, want own non-admin message", userMessage)
	}

	response = supportJSONRequest(t, handler, http.MethodGet, "/api/admin/support/conversations", adminToken, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("admin conversations status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	var conversations []SupportConversationOut
	decodeSupportResponse(t, response, &conversations)
	if len(conversations) != 1 {
		t.Fatalf("admin conversations length = %d, want 1", len(conversations))
	}
	conversation := conversations[0]
	if conversation.User.ID != officer.ID || conversation.LastMessage != "Нужна помощь" || conversation.MessageCount != 1 {
		t.Fatalf("admin conversation = %#v, want officer conversation with last message", conversation)
	}

	adminMessagesPath := "/api/admin/support/conversations/" + intString(conversation.ID) + "/messages"
	response = supportJSONRequest(t, handler, http.MethodGet, adminMessagesPath, officerToken, nil)
	if response.Code != http.StatusForbidden {
		t.Fatalf("user admin-message access status = %d, want %d", response.Code, http.StatusForbidden)
	}

	response = supportJSONRequest(t, handler, http.MethodPost, adminMessagesPath, adminToken, map[string]string{"body": "Ответ администратора"})
	if response.Code != http.StatusOK {
		t.Fatalf("admin reply status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	var adminReply SupportMessageOut
	decodeSupportResponse(t, response, &adminReply)
	if adminReply.Body != "Ответ администратора" || !adminReply.SentByMe || !adminReply.SentByAdmin {
		t.Fatalf("admin reply = %#v, want own admin reply", adminReply)
	}

	response = supportJSONRequest(t, handler, http.MethodGet, "/api/support/conversation", officerToken, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("user refreshed conversation status = %d, want %d", response.Code, http.StatusOK)
	}
	var refreshed SupportThreadOut
	decodeSupportResponse(t, response, &refreshed)
	if refreshed.Conversation == nil || refreshed.Conversation.ID != conversation.ID {
		t.Fatalf("refreshed conversation = %#v, want conversation %d", refreshed.Conversation, conversation.ID)
	}
	if len(refreshed.Messages) != 2 {
		t.Fatalf("refreshed messages length = %d, want 2", len(refreshed.Messages))
	}
	if refreshed.Messages[1].Body != "Ответ администратора" || refreshed.Messages[1].SentByMe || !refreshed.Messages[1].SentByAdmin {
		t.Fatalf("user sees admin reply = %#v, want received admin reply", refreshed.Messages[1])
	}

	response = supportJSONRequest(t, handler, http.MethodGet, "/api/support/conversation", otherOfficerToken, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("other user conversation status = %d, want %d", response.Code, http.StatusOK)
	}
	var otherThread SupportThreadOut
	decodeSupportResponse(t, response, &otherThread)
	if otherThread.Conversation != nil || len(otherThread.Messages) != 0 {
		t.Fatalf("other user thread = %#v, want isolated empty chat", otherThread)
	}
}

func TestNormalizeSupportBodyCountsCharacters(t *testing.T) {
	body := strings.Repeat("ә", supportMaxBodyLength)
	normalized, err := normalizeSupportBody(body)
	if err != nil {
		t.Fatalf("normalize exact unicode limit: %v", err)
	}
	if normalized != body {
		t.Fatalf("normalized exact unicode body changed")
	}

	_, err = normalizeSupportBody(body + "ә")
	var hErr httpErr
	if !errors.As(err, &hErr) || hErr.status != http.StatusBadRequest {
		t.Fatalf("normalize over unicode limit error = %v, want bad request", err)
	}
}

func newSupportTestApp(t *testing.T) *App {
	t.Helper()
	dbPath := filepath.ToSlash(filepath.Join(t.TempDir(), "kpi_support.db"))
	t.Setenv("KPI_DATABASE_URL", "sqlite:///"+dbPath)
	t.Setenv("KPI_SEED_MODE", "")
	t.Setenv("KPI_UPLOAD_DIR", filepath.Join(t.TempDir(), "uploads"))
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	return app
}

func insertSupportDistrict(t *testing.T, app *App) int {
	t.Helper()
	var districtID int
	err := app.db.QueryRowContext(context.Background(), `INSERT INTO districts(name, coeff)
		VALUES('Support District', 1.0)
		RETURNING id`).Scan(&districtID)
	if err != nil {
		t.Fatal(err)
	}
	return districtID
}

func insertSupportUser(t *testing.T, app *App, username string, fullName string, rank string, role string, districtID *int) *User {
	t.Helper()
	passwordHash, err := hashPassword("pass123")
	if err != nil {
		t.Fatal(err)
	}
	var userID int
	err = app.db.QueryRowContext(context.Background(), `INSERT INTO users
		(username, password_hash, full_name, rank, role, district_id, monthly_norm)
		VALUES (?, ?, ?, ?, ?, ?, 100)
		RETURNING id`,
		username,
		passwordHash,
		fullName,
		rank,
		role,
		nullableInt(districtID),
	).Scan(&userID)
	if err != nil {
		t.Fatal(err)
	}
	user, err := app.getUserByID(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	return user
}

func supportToken(t *testing.T, app *App, user *User) string {
	t.Helper()
	token, err := app.createToken(user)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func supportJSONRequest(t *testing.T, handler http.Handler, method string, path string, token string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			t.Fatal(err)
		}
	}
	request := httptest.NewRequest(method, path, &body)
	request.Header.Set("Authorization", "Bearer "+token)
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func decodeSupportResponse(t *testing.T, response *httptest.ResponseRecorder, dest any) {
	t.Helper()
	if err := json.NewDecoder(response.Body).Decode(dest); err != nil {
		t.Fatal(err)
	}
}

func intString(value int) string {
	return strconv.Itoa(value)
}
