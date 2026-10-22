package main

import (
	"errors"
	"net/http"

	domainnotifications "kpi-police/backend/internal/domain/notifications"
)

func (a *App) handleListNotifications(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(a, w, r)
	if !ok {
		return
	}
	limit := 0
	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := parseInt(value)
		if err != nil {
			writeHTTPError(w, notificationHTTPError(domainnotifications.ErrInvalidLimit))
			return
		}
		limit = parsed
	}
	list, err := a.notificationService().ListNotifications(r.Context(), user.ID, limit)
	if err != nil {
		writeHTTPError(w, notificationHTTPError(err))
		return
	}
	writeJSON(w, http.StatusOK, notificationListOut(list))
}

func (a *App) handleMarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(a, w, r)
	if !ok {
		return
	}
	notificationID, valid := parseIDPathValue(r, "notification_id")
	if !valid {
		writeError(w, http.StatusNotFound, "Уведомление не найдено")
		return
	}
	item, err := a.notificationService().MarkRead(r.Context(), user.ID, notificationID)
	if err != nil {
		writeHTTPError(w, notificationHTTPError(err))
		return
	}
	writeJSON(w, http.StatusOK, notificationOut(item))
}

func (a *App) handleMarkAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(a, w, r)
	if !ok {
		return
	}
	updated, err := a.notificationService().MarkAllRead(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"updated": updated})
}

func notificationListOut(list domainnotifications.List) NotificationListOut {
	return NotificationListOut{
		UnreadCount: list.UnreadCount,
		Items:       notificationItemsOut(list.Items),
	}
}

func notificationItemsOut(items []domainnotifications.Notification) []NotificationOut {
	result := make([]NotificationOut, 0, len(items))
	for _, item := range items {
		result = append(result, notificationOut(item))
	}
	return result
}

func notificationOut(item domainnotifications.Notification) NotificationOut {
	return NotificationOut{
		ID:        item.ID,
		Kind:      item.Kind,
		Title:     item.Title,
		Message:   item.Message,
		Link:      item.Link,
		IsRead:    item.IsRead,
		CreatedAt: item.CreatedAt,
	}
}

func notificationHTTPError(err error) error {
	switch {
	case errors.Is(err, domainnotifications.ErrInvalidLimit):
		return httpErr{status: http.StatusBadRequest, detail: "Некорректный лимит"}
	case errors.Is(err, domainnotifications.ErrNotFound):
		return httpErr{status: http.StatusNotFound, detail: "Уведомление не найдено"}
	default:
		return err
	}
}
