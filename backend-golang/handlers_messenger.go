package main

import (
	"errors"
	"net/http"

	domainmessenger "kpi-police/backend-golang/internal/domain/messenger"
	domainusers "kpi-police/backend-golang/internal/domain/users"
)

func (a *App) handleMessengerContacts(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(a, w, r)
	if !ok {
		return
	}
	contacts, err := a.messengerService().ListContacts(r.Context(), domainUser(user))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	efficiency := map[int]domainusers.Efficiency{}
	includeEfficiency := r.URL.Query().Get("include_efficiency")
	if includeEfficiency != "0" && includeEfficiency != "false" {
		scope, _ := headScopeDistrictValue(user)
		efficiency, err = a.userQueryService().EfficiencyMap(r.Context(), scope)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, messengerContactsOut(contacts, efficiency))
}

func (a *App) handleMessengerUnreadCount(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(a, w, r)
	if !ok {
		return
	}
	count, err := a.messengerService().UnreadCount(r.Context(), domainUser(user))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, MessengerUnreadOut{UnreadCount: count})
}

func (a *App) handleMessengerEntries(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(a, w, r)
	if !ok {
		return
	}
	contactID, valid := parseIDPathValue(r, "contact_id")
	if !valid {
		writeError(w, http.StatusNotFound, "Пользователь не найден")
		return
	}
	entries, err := a.messengerService().ListEntries(r.Context(), domainUser(user), contactID)
	if err != nil {
		writeHTTPError(w, messengerContactHTTPError(err))
		return
	}
	writeJSON(w, http.StatusOK, messengerEntriesOut(entries))
}

func (a *App) handleMessengerMessages(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(a, w, r)
	if !ok {
		return
	}
	contactID, valid := parseIDPathValue(r, "contact_id")
	if !valid {
		writeError(w, http.StatusNotFound, "Пользователь не найден")
		return
	}
	afterID, err := parseOptionalInt(r.URL.Query().Get("after_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Сообщение указано неверно")
		return
	}
	var messages []domainmessenger.Message
	if afterID != nil {
		messages, err = a.messengerService().ConversationMessagesAfter(r.Context(), domainUser(user), contactID, *afterID)
	} else {
		messages, err = a.messengerService().ConversationMessages(r.Context(), domainUser(user), contactID)
	}
	if err != nil {
		writeHTTPError(w, messengerContactHTTPError(err))
		return
	}
	writeJSON(w, http.StatusOK, messagesOut(messages, user.ID))
}

func (a *App) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(a, w, r)
	if !ok {
		return
	}
	var payload MessageCreate
	if err := readJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Некорректные данные сообщения")
		return
	}
	message, err := a.messengerService().SendMessage(r.Context(), domainUser(user), domainmessenger.SendRequest{
		RecipientID: payload.RecipientID,
		Body:        payload.Body,
		EntryID:     payload.EntryID,
	})
	if err != nil {
		writeHTTPError(w, messengerSendHTTPError(err))
		return
	}
	writeJSON(w, http.StatusOK, messageOut(message, user.ID))
}

func (a *App) handleDeleteMessage(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(a, w, r)
	if !ok {
		return
	}
	messageID, valid := parseIDPathValue(r, "message_id")
	if !valid {
		writeError(w, http.StatusNotFound, "Сообщение не найдено")
		return
	}
	message, err := a.messengerService().DeleteMessage(r.Context(), domainUser(user), messageID)
	if err != nil {
		writeHTTPError(w, messengerDeleteHTTPError(err))
		return
	}
	writeJSON(w, http.StatusOK, messageOut(message, user.ID))
}

func domainUser(user *User) domainusers.User {
	result := domainusers.User{
		ID:          user.ID,
		Username:    user.Username,
		FullName:    user.FullName,
		Rank:        user.Rank,
		Role:        user.Role,
		MonthlyNorm: user.MonthlyNorm,
	}
	if user.DistrictID.Valid {
		id := int(user.DistrictID.Int64)
		result.DistrictID = &id
	}
	if user.District != nil {
		district := domainusers.District{
			ID:    user.District.ID,
			Name:  user.District.Name,
			Coeff: user.District.Coeff,
		}
		result.District = &district
	}
	return result
}

func messengerContactsOut(contacts []domainmessenger.Contact, efficiency map[int]domainusers.Efficiency) []MessengerContactOut {
	result := make([]MessengerContactOut, 0, len(contacts))
	for _, contact := range contacts {
		row := MessengerContactOut{
			ID:                 contact.User.ID,
			FullName:           contact.User.FullName,
			Rank:               contact.User.Rank,
			Role:               contact.User.Role,
			DistrictName:       contact.User.DistrictName,
			UnreadCount:        contact.UnreadCount,
			LastMessage:        contact.LastMessage,
			LastMessageAt:      contact.LastMessageAt,
			LastMessageDeleted: contact.LastMessageDeleted,
		}
		if info, ok := efficiency[contact.User.ID]; ok {
			rank := info.Rank
			total := info.Total
			score := info.Score
			row.EfficiencyRank = &rank
			row.EfficiencyTotal = &total
			row.EfficiencyScore = &score
		}
		result = append(result, row)
	}
	return result
}

func messengerEntriesOut(entries []domainmessenger.Entry) []MessengerEntryOut {
	result := make([]MessengerEntryOut, 0, len(entries))
	for _, entry := range entries {
		result = append(result, messengerEntryDomainOut(entry))
	}
	return result
}

func messengerEntryDomainOut(entry domainmessenger.Entry) MessengerEntryOut {
	return MessengerEntryOut{
		ID:              entry.ID,
		Date:            entry.Date,
		Code:            entry.Code,
		Indicator:       entry.Indicator,
		Block:           entry.Block,
		NPA:             entry.NPA,
		SourceURL:       entry.SourceURL,
		Quantity:        round2(entry.Quantity),
		CountedQuantity: round2(entry.CountedQuantity),
		MonthlyLimit:    round2(entry.MonthlyLimit),
		PointsPerUnit:   round2(entry.PointsPerUnit),
		Coeff:           round2(entry.Coeff),
		RawScore:        round2(entry.RawScore),
		CappedScore:     round2(entry.CappedScore),
		AntiCoeff:       round2(entry.AntiCoeff),
		FinalScore:      round2(entry.FinalScore),
		Status:          entry.Status,
		Comments:        entry.Comments,
		HeadComment:     entry.HeadComment,
	}
}

func messagesOut(messages []domainmessenger.Message, userID int) []MessageOut {
	result := make([]MessageOut, 0, len(messages))
	for _, message := range messages {
		result = append(result, messageOut(message, userID))
	}
	return result
}

func messageOut(message domainmessenger.Message, userID int) MessageOut {
	isDeleted := message.DeletedAt != nil
	body := message.Body
	var entry *MessengerEntryOut
	if isDeleted {
		body = ""
	} else if message.Entry != nil {
		out := messengerEntryDomainOut(*message.Entry)
		entry = &out
	}
	return MessageOut{
		ID:          message.ID,
		SenderID:    message.SenderID,
		RecipientID: message.RecipientID,
		SenderName:  message.SenderName,
		Body:        body,
		CreatedAt:   message.CreatedAt,
		ReadAt:      message.ReadAt,
		SentByMe:    message.SenderID == userID,
		IsDeleted:   isDeleted,
		Entry:       entry,
	}
}

func messengerContactHTTPError(err error) error {
	switch {
	case errors.Is(err, domainmessenger.ErrContactNotFound):
		return httpErr{status: http.StatusNotFound, detail: "Пользователь не найден"}
	case errors.Is(err, domainmessenger.ErrForbidden):
		return httpErr{status: http.StatusForbidden, detail: "Этот диалог недоступен"}
	default:
		return err
	}
}

func messengerSendHTTPError(err error) error {
	switch {
	case errors.Is(err, domainmessenger.ErrContactNotFound):
		return httpErr{status: http.StatusNotFound, detail: "Пользователь не найден"}
	case errors.Is(err, domainmessenger.ErrForbidden):
		return httpErr{status: http.StatusForbidden, detail: "Этот диалог недоступен"}
	case errors.Is(err, domainmessenger.ErrEmptyBody):
		return httpErr{status: http.StatusBadRequest, detail: "Сообщение не может быть пустым"}
	case errors.Is(err, domainmessenger.ErrBodyTooLong):
		return httpErr{status: http.StatusBadRequest, detail: "Сообщение слишком длинное"}
	case errors.Is(err, domainmessenger.ErrEntryNotFound):
		return httpErr{status: http.StatusNotFound, detail: "Запись KPI для этого диалога не найдена"}
	default:
		return err
	}
}

func messengerDeleteHTTPError(err error) error {
	switch {
	case errors.Is(err, domainmessenger.ErrMessageNotFound):
		return httpErr{status: http.StatusNotFound, detail: "Сообщение не найдено"}
	case errors.Is(err, domainmessenger.ErrForbidden):
		return httpErr{status: http.StatusForbidden, detail: "Можно удалить только своё сообщение"}
	default:
		return err
	}
}
