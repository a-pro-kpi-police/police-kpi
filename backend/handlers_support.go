package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"
)

const supportMaxBodyLength = 2000

type supportMessageRecord struct {
	ID             int
	ConversationID int
	SenderID       int
	SenderName     string
	SenderRole     string
	Body           string
	CreatedAt      time.Time
}

func (a *App) handleSupportConversation(w http.ResponseWriter, r *http.Request) {
	user, ok := requireSupportUser(a, w, r)
	if !ok {
		return
	}
	conversation, found, err := a.supportConversationForUser(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !found {
		writeJSON(w, http.StatusOK, SupportThreadOut{Messages: []SupportMessageOut{}})
		return
	}
	messages, err := a.supportMessages(r.Context(), conversation.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, supportThreadOut(&conversation, messages, user.ID))
}

func (a *App) handleSupportSendMessage(w http.ResponseWriter, r *http.Request) {
	user, ok := requireSupportUser(a, w, r)
	if !ok {
		return
	}
	var payload SupportMessageCreate
	if err := readJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Некорректные данные сообщения")
		return
	}
	message, err := a.createUserSupportMessage(r.Context(), user.ID, payload.Body)
	if err != nil {
		writeHTTPError(w, supportMessageHTTPError(err))
		return
	}
	writeJSON(w, http.StatusOK, supportMessageOut(message, user.ID))
}

func (a *App) handleAdminSupportConversations(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAdmin(a, w, r); !ok {
		return
	}
	conversations, err := a.supportConversations(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, conversations)
}

func (a *App) handleAdminSupportMessages(w http.ResponseWriter, r *http.Request) {
	admin, ok := requireAdmin(a, w, r)
	if !ok {
		return
	}
	conversationID, valid := parseIDPathValue(r, "conversation_id")
	if !valid {
		writeError(w, http.StatusNotFound, "Диалог поддержки не найден")
		return
	}
	conversation, err := a.supportConversationByID(r.Context(), conversationID)
	if err != nil {
		writeHTTPError(w, supportConversationHTTPError(err))
		return
	}
	messages, err := a.supportMessages(r.Context(), conversation.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, supportThreadOut(&conversation, messages, admin.ID))
}

func (a *App) handleAdminSupportSendMessage(w http.ResponseWriter, r *http.Request) {
	admin, ok := requireAdmin(a, w, r)
	if !ok {
		return
	}
	conversationID, valid := parseIDPathValue(r, "conversation_id")
	if !valid {
		writeError(w, http.StatusNotFound, "Диалог поддержки не найден")
		return
	}
	conversation, err := a.supportConversationByID(r.Context(), conversationID)
	if err != nil {
		writeHTTPError(w, supportConversationHTTPError(err))
		return
	}
	var payload SupportMessageCreate
	if err := readJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Некорректные данные сообщения")
		return
	}
	message, err := a.createSupportMessage(r.Context(), conversation.ID, admin.ID, payload.Body)
	if err != nil {
		writeHTTPError(w, supportMessageHTTPError(err))
		return
	}
	writeJSON(w, http.StatusOK, supportMessageOut(message, admin.ID))
}

func requireSupportUser(a *App, w http.ResponseWriter, r *http.Request) (*User, bool) {
	user, ok := requireUser(a, w, r)
	if !ok {
		return nil, false
	}
	if user.Role == roleAdmin {
		writeError(w, http.StatusForbidden, "Чат поддержки доступен только сотрудникам")
		return nil, false
	}
	return user, true
}

func (a *App) supportConversations(ctx context.Context) ([]SupportConversationOut, error) {
	rows, err := a.db.QueryContext(ctx, supportConversationSelect+`
		ORDER BY COALESCE((
			SELECT m.created_at
			FROM support_messages m
			WHERE m.conversation_id = c.id
			ORDER BY m.created_at DESC, m.id DESC
			LIMIT 1
		), c.updated_at) DESC, c.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	conversations := []SupportConversationOut{}
	for rows.Next() {
		conversation, err := scanSupportConversation(rows)
		if err != nil {
			return nil, err
		}
		conversations = append(conversations, conversation)
	}
	return conversations, rows.Err()
}

func (a *App) supportConversationForUser(ctx context.Context, userID int) (SupportConversationOut, bool, error) {
	conversation, err := scanSupportConversation(a.db.QueryRowContext(ctx, supportConversationSelect+` WHERE c.user_id = ?`, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return SupportConversationOut{}, false, nil
	}
	return conversation, err == nil, err
}

func (a *App) supportConversationByID(ctx context.Context, conversationID int) (SupportConversationOut, error) {
	conversation, err := scanSupportConversation(a.db.QueryRowContext(ctx, supportConversationSelect+` WHERE c.id = ?`, conversationID))
	if errors.Is(err, sql.ErrNoRows) {
		return SupportConversationOut{}, httpErr{status: http.StatusNotFound, detail: "Диалог поддержки не найден"}
	}
	return conversation, err
}

func (a *App) supportMessages(ctx context.Context, conversationID int) ([]supportMessageRecord, error) {
	rows, err := a.db.QueryContext(ctx, supportMessageSelect+` WHERE m.conversation_id = ? ORDER BY m.created_at ASC, m.id ASC`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []supportMessageRecord{}
	for rows.Next() {
		message, err := scanSupportMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, rows.Err()
}

func (a *App) createUserSupportMessage(ctx context.Context, userID int, rawBody string) (supportMessageRecord, error) {
	body, err := normalizeSupportBody(rawBody)
	if err != nil {
		return supportMessageRecord{}, err
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return supportMessageRecord{}, err
	}
	defer tx.Rollback()

	now := nowDB()
	if _, err := tx.ExecContext(ctx, `INSERT INTO support_conversations
		(user_id, created_at, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id) DO NOTHING`,
		userID, now, now,
	); err != nil {
		return supportMessageRecord{}, err
	}

	var conversationID int
	if err := tx.QueryRowContext(ctx, `SELECT id FROM support_conversations WHERE user_id = ?`, userID).Scan(&conversationID); err != nil {
		return supportMessageRecord{}, err
	}
	messageID, err := insertSupportMessage(ctx, tx, conversationID, userID, body, now)
	if err != nil {
		return supportMessageRecord{}, err
	}
	if err := tx.Commit(); err != nil {
		return supportMessageRecord{}, err
	}
	return a.supportMessageByID(ctx, messageID)
}

func (a *App) createSupportMessage(ctx context.Context, conversationID int, senderID int, rawBody string) (supportMessageRecord, error) {
	body, err := normalizeSupportBody(rawBody)
	if err != nil {
		return supportMessageRecord{}, err
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return supportMessageRecord{}, err
	}
	defer tx.Rollback()

	now := nowDB()
	messageID, err := insertSupportMessage(ctx, tx, conversationID, senderID, body, now)
	if err != nil {
		return supportMessageRecord{}, err
	}
	if err := tx.Commit(); err != nil {
		return supportMessageRecord{}, err
	}
	return a.supportMessageByID(ctx, messageID)
}

func insertSupportMessage(ctx context.Context, tx dbtx, conversationID int, senderID int, body string, createdAt string) (int, error) {
	var messageID int
	err := tx.QueryRowContext(ctx, `INSERT INTO support_messages
		(conversation_id, sender_id, body, created_at)
		VALUES (?, ?, ?, ?)
		RETURNING id`,
		conversationID, senderID, body, createdAt,
	).Scan(&messageID)
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE support_conversations SET updated_at = ? WHERE id = ?`, createdAt, conversationID); err != nil {
		return 0, err
	}
	return messageID, nil
}

func (a *App) supportMessageByID(ctx context.Context, messageID int) (supportMessageRecord, error) {
	message, err := scanSupportMessage(a.db.QueryRowContext(ctx, supportMessageSelect+` WHERE m.id = ?`, messageID))
	if errors.Is(err, sql.ErrNoRows) {
		return supportMessageRecord{}, httpErr{status: http.StatusNotFound, detail: "Сообщение не найдено"}
	}
	return message, err
}

func normalizeSupportBody(raw string) (string, error) {
	body := strings.TrimSpace(raw)
	if body == "" {
		return "", httpErr{status: http.StatusBadRequest, detail: "Сообщение не может быть пустым"}
	}
	if len([]rune(body)) > supportMaxBodyLength {
		return "", httpErr{status: http.StatusBadRequest, detail: "Сообщение слишком длинное"}
	}
	return body, nil
}

func supportThreadOut(conversation *SupportConversationOut, messages []supportMessageRecord, viewerID int) SupportThreadOut {
	return SupportThreadOut{
		Conversation: conversation,
		Messages:     supportMessagesOut(messages, viewerID),
	}
}

func supportMessagesOut(messages []supportMessageRecord, viewerID int) []SupportMessageOut {
	result := make([]SupportMessageOut, 0, len(messages))
	for _, message := range messages {
		result = append(result, supportMessageOut(message, viewerID))
	}
	return result
}

func supportMessageOut(message supportMessageRecord, viewerID int) SupportMessageOut {
	return SupportMessageOut{
		ID:             message.ID,
		ConversationID: message.ConversationID,
		SenderID:       message.SenderID,
		SenderName:     message.SenderName,
		SenderRole:     message.SenderRole,
		Body:           message.Body,
		CreatedAt:      message.CreatedAt,
		SentByMe:       message.SenderID == viewerID,
		SentByAdmin:    message.SenderRole == roleAdmin,
	}
}

func scanSupportConversation(scanner sqlScanner) (SupportConversationOut, error) {
	var conversation SupportConversationOut
	var districtID sql.NullInt64
	var districtName sql.NullString
	var districtCoeff sql.NullFloat64
	var createdAt sql.NullString
	var updatedAt sql.NullString
	var lastMessage sql.NullString
	var lastMessageAt sql.NullString
	if err := scanner.Scan(
		&conversation.ID,
		&conversation.User.ID,
		&conversation.User.Username,
		&conversation.User.FullName,
		&conversation.User.Rank,
		&conversation.User.Role,
		&conversation.User.MonthlyNorm,
		&districtID,
		&districtName,
		&districtCoeff,
		&createdAt,
		&updatedAt,
		&lastMessage,
		&lastMessageAt,
		&conversation.MessageCount,
	); err != nil {
		return SupportConversationOut{}, err
	}
	conversation.User.MonthlyNorm = round2(conversation.User.MonthlyNorm)
	if districtID.Valid {
		conversation.User.District = &District{
			ID:    int(districtID.Int64),
			Name:  districtName.String,
			Coeff: districtCoeff.Float64,
		}
	}
	conversation.CreatedAt = fallbackTime(createdAt)
	conversation.UpdatedAt = fallbackTime(updatedAt)
	conversation.LastMessage = lastMessage.String
	conversation.LastMessageAt = parseDBTime(lastMessageAt)
	return conversation, nil
}

func scanSupportMessage(scanner sqlScanner) (supportMessageRecord, error) {
	var message supportMessageRecord
	var createdAt sql.NullString
	if err := scanner.Scan(
		&message.ID,
		&message.ConversationID,
		&message.SenderID,
		&message.SenderName,
		&message.SenderRole,
		&message.Body,
		&createdAt,
	); err != nil {
		return supportMessageRecord{}, err
	}
	message.CreatedAt = fallbackTime(createdAt)
	return message, nil
}

func fallbackTime(value sql.NullString) time.Time {
	if parsed := parseDBTime(value); parsed != nil {
		return *parsed
	}
	return time.Now().UTC()
}

func supportConversationHTTPError(err error) error {
	var hErr httpErr
	if errors.As(err, &hErr) {
		return hErr
	}
	return err
}

func supportMessageHTTPError(err error) error {
	var hErr httpErr
	if errors.As(err, &hErr) {
		return hErr
	}
	return err
}

const supportConversationSelect = `SELECT
	c.id,
	u.id, u.username, u.full_name, u.rank, u.role, u.monthly_norm,
	d.id, d.name, d.coeff,
	c.created_at, c.updated_at,
	(SELECT m.body
		FROM support_messages m
		WHERE m.conversation_id = c.id
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT 1) AS last_message,
	(SELECT m.created_at
		FROM support_messages m
		WHERE m.conversation_id = c.id
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT 1) AS last_message_at,
	(SELECT COUNT(*)
		FROM support_messages m
		WHERE m.conversation_id = c.id) AS message_count
FROM support_conversations c
JOIN users u ON u.id = c.user_id
LEFT JOIN districts d ON d.id = u.district_id`

const supportMessageSelect = `SELECT
	m.id, m.conversation_id, m.sender_id,
	u.full_name, u.role,
	m.body, m.created_at
FROM support_messages m
JOIN users u ON u.id = m.sender_id`
