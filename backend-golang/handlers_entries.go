package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	appentries "kpi-police/backend-golang/internal/application/entries"
	domainentries "kpi-police/backend-golang/internal/domain/entries"
)

const (
	maxAttachmentSize        = 20 * 1024 * 1024
	maxAttachmentsPerRequest = 5
	attachmentPlaybackTTL    = time.Hour
)

var allowedAttachmentExtensions = map[string]bool{
	".csv": true, ".doc": true, ".docx": true, ".jpeg": true, ".jpg": true,
	".pdf": true, ".png": true, ".txt": true, ".webp": true, ".xls": true,
	".xlsx": true, ".zip": true,
	".mp4": true, ".ogv": true, ".webm": true,
}

var attachmentContentTypes = map[string]string{
	".jpeg": "image/jpeg", ".jpg": "image/jpeg", ".png": "image/png", ".webp": "image/webp",
	".mp4": "video/mp4", ".ogv": "video/ogg", ".webm": "video/webm",
}

func (a *App) handleListEntries(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(a, w, r)
	if !ok {
		return
	}
	queryValues := r.URL.Query()
	entryQuery := domainentries.EntryQuery{Actor: entryActor(user)}
	if isEntryManagerRole(user.Role) {
		if officerText := queryValues.Get("officer_id"); officerText != "" {
			officerID, err := parseInt(officerText)
			if err != nil {
				writeError(w, http.StatusBadRequest, "Сотрудник указан неверно")
				return
			}
			entryQuery.OfficerID = &officerID
		}
		if scope, scoped := headScopeDistrictValue(user); scoped {
			entryQuery.DistrictID = &scope
		}
	}
	if month := strings.TrimSpace(queryValues.Get("month")); month != "" {
		entryQuery.Month = month
	} else if periodType, periodValue := strings.TrimSpace(queryValues.Get("period_type")), strings.TrimSpace(queryValues.Get("period_value")); periodType != "" && periodValue != "" {
		switch periodType {
		case "month":
			entryQuery.Month = periodValue
		case "quarter":
			months, err := quarterMonths(periodValue)
			if err != nil {
				writeError(w, http.StatusBadRequest, "Некорректный квартал")
				return
			}
			entryQuery.Months = []string{months[0], months[1], months[2]}
		case "year":
			if len(periodValue) != 4 {
				writeError(w, http.StatusBadRequest, "Некорректный год")
				return
			}
			entryQuery.Year = periodValue
		default:
			writeError(w, http.StatusBadRequest, "Некорректный тип периода")
			return
		}
	}
	if status := strings.TrimSpace(queryValues.Get("status")); status != "" {
		entryQuery.Status = status
	}

	entryQuery.SortKey = queryValues.Get("sort_key")
	entryQuery.SortAscending = queryValues.Get("sort_dir") == "asc"
	if offsetText := queryValues.Get("offset"); offsetText != "" {
		offset, err := parseInt(offsetText)
		if err != nil || offset < 0 {
			writeError(w, http.StatusBadRequest, "Некорректное смещение")
			return
		}
		entryQuery.Offset = &offset
		if limitText := queryValues.Get("limit"); limitText != "" {
			limit, err := parseInt(limitText)
			if err != nil || limit < 1 || limit > 100 {
				writeError(w, http.StatusBadRequest, "Некорректный лимит")
				return
			}
			entryQuery.Limit = &limit
		}
	} else if limitText := queryValues.Get("limit"); limitText != "" {
		limit, err := parseInt(limitText)
		if err != nil || limit < 1 || limit > 100 {
			writeError(w, http.StatusBadRequest, "Некорректный лимит")
			return
		}
		entryQuery.Limit = &limit
	}

	entries, err := a.entryQueryService().ListEntries(r.Context(), entryQuery)
	if err != nil {
		writeHTTPError(w, entryQueryHTTPError(err))
		return
	}
	writeJSON(w, http.StatusOK, listedEntriesOut(entries))
}

func entryQueryHTTPError(err error) error {
	switch {
	case errors.Is(err, domainentries.ErrInvalidOffset):
		return httpErr{status: http.StatusBadRequest, detail: "Некорректное смещение"}
	case errors.Is(err, domainentries.ErrInvalidLimit):
		return httpErr{status: http.StatusBadRequest, detail: "Некорректный лимит"}
	default:
		return err
	}
}

func (a *App) handleCreateEntry(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(a, w, r)
	if !ok {
		return
	}
	var payload EntryCreate
	if err := readJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Некорректные данные записи")
		return
	}
	created, err := a.createKpiEntry(r.Context(), user, payload, nil)
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, listedEntryOut(created))
}

func (a *App) handleCreateEntryWithAttachment(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(a, w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAttachmentSize*maxAttachmentsPerRequest+2*1024*1024)
	if err := r.ParseMultipartForm(8 * 1024 * 1024); err != nil {
		writeError(w, http.StatusBadRequest, "Некорректная форма вложения")
		return
	}
	defer r.MultipartForm.RemoveAll()
	catalogID, err := parseInt(r.FormValue("catalog_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Показатель KPI не найден")
		return
	}
	officerID, err := parseOptionalInt(r.FormValue("officer_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Сотрудник указан неверно")
		return
	}
	quantity, err := parseFloat(r.FormValue("quantity"), 1)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Количество указано неверно")
		return
	}
	var entryDate *string
	if value := strings.TrimSpace(r.FormValue("date")); value != "" {
		entryDate = &value
	}
	payload := EntryCreate{
		CatalogID: catalogID,
		OfficerID: officerID,
		Quantity:  &quantity,
		Date:      entryDate,
		Comments:  r.FormValue("comments"),
	}
	attachments, err := a.storeAttachments(r)
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	created, err := a.createKpiEntry(r.Context(), user, payload, domainAttachments(attachments))
	if err != nil {
		a.removeUploadedAttachments(attachments)
		writeHTTPError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, listedEntryOut(created))
}

func (a *App) handleDownloadAttachment(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(a, w, r)
	if !ok {
		return
	}
	entryID, valid := parseIDPathValue(r, "entry_id")
	if !valid {
		writeError(w, http.StatusNotFound, "Запись не найдена")
		return
	}
	entry, err := a.entryQueryService().GetEntry(r.Context(), entryID)
	if errors.Is(err, domainentries.ErrEntryNotFound) {
		writeError(w, http.StatusNotFound, "Запись не найдена")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !isEntryManagerRole(user.Role) && entry.OfficerID != user.ID {
		writeError(w, http.StatusForbidden, "Недостаточно прав")
		return
	}
	if err := a.ensureOfficerInHeadScope(r.Context(), user, entry.OfficerID); err != nil {
		writeHTTPError(w, err)
		return
	}
	if entry.AttachmentStorageName == "" || entry.AttachmentName == "" {
		writeError(w, http.StatusNotFound, "Вложение не найдено")
		return
	}
	if strings.ContainsAny(entry.AttachmentStorageName, `/\`) || filepath.Base(entry.AttachmentStorageName) != entry.AttachmentStorageName {
		writeError(w, http.StatusNotFound, "Вложение не найдено")
		return
	}
	storagePath := filepath.Join(a.uploadDir, entry.AttachmentStorageName)
	if _, err := os.Stat(storagePath); err != nil {
		writeError(w, http.StatusNotFound, "Вложение не найдено")
		return
	}
	a.serveStoredAttachment(w, r, storagePath, entry.AttachmentName, entry.AttachmentContentType, true)
}

func (a *App) handleAddEntryAttachments(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(a, w, r)
	if !ok {
		return
	}
	entryID, valid := parseIDPathValue(r, "entry_id")
	if !valid {
		writeError(w, http.StatusNotFound, "Запись не найдена")
		return
	}
	entry, err := a.authorizedEntry(r, user, entryID)
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAttachmentSize*maxAttachmentsPerRequest+2*1024*1024)
	if err := r.ParseMultipartForm(8 * 1024 * 1024); err != nil {
		writeError(w, http.StatusBadRequest, "Некорректная форма вложения")
		return
	}
	defer r.MultipartForm.RemoveAll()
	attachments, err := a.storeAttachments(r)
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	if err := a.insertEntryAttachments(r.Context(), entry.ID, attachments, entry.AttachmentStorageName == ""); err != nil {
		a.removeUploadedAttachments(attachments)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	updated, err := a.entryQueryService().GetEntry(r.Context(), entry.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, listedEntryOut(updated))
}

func (a *App) handleEntryAttachment(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(a, w, r)
	if !ok {
		return
	}
	_, attachment, err := a.authorizedAttachment(r, user)
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	storagePath, ok := a.attachmentStoragePath(attachment.StorageName)
	if !ok {
		writeError(w, http.StatusNotFound, "Вложение не найдено")
		return
	}
	a.serveStoredAttachment(w, r, storagePath, attachment.Name, attachment.ContentType, true)
}

func (a *App) handleAttachmentPlaybackURL(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(a, w, r)
	if !ok {
		return
	}
	entry, attachment, err := a.authorizedAttachment(r, user)
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	if !strings.HasPrefix(normalizedAttachmentContentType(attachment.Name, attachment.ContentType), "video/") {
		writeError(w, http.StatusBadRequest, "Недопустимый тип файла")
		return
	}
	token := a.createAttachmentPlaybackToken(entry.ID, attachment.ID)
	writeJSON(w, http.StatusOK, AttachmentPlaybackOut{URL: fmt.Sprintf(
		"/api/entries/%d/attachments/%d/playback?token=%s", entry.ID, attachment.ID, token,
	)})
}

func (a *App) handleAttachmentPlayback(w http.ResponseWriter, r *http.Request) {
	entryID, validEntry := parseIDPathValue(r, "entry_id")
	attachmentID, validAttachment := parseIDPathValue(r, "attachment_id")
	if !validEntry || !validAttachment || !a.verifyAttachmentPlaybackToken(r.URL.Query().Get("token"), entryID, attachmentID) {
		writeError(w, http.StatusUnauthorized, "Не удалось проверить учётные данные")
		return
	}
	entry, err := a.entryQueryService().GetEntry(r.Context(), entryID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Вложение не найдено")
		return
	}
	attachment, ok := findListedAttachment(entry, attachmentID)
	if !ok {
		writeError(w, http.StatusNotFound, "Вложение не найдено")
		return
	}
	contentType := normalizedAttachmentContentType(attachment.Name, attachment.ContentType)
	if !strings.HasPrefix(contentType, "video/") {
		writeError(w, http.StatusBadRequest, "Недопустимый тип файла")
		return
	}
	storagePath, ok := a.attachmentStoragePath(attachment.StorageName)
	if !ok {
		writeError(w, http.StatusNotFound, "Вложение не найдено")
		return
	}
	a.serveStoredAttachment(w, r, storagePath, attachment.Name, contentType, false)
}

func (a *App) authorizedEntry(r *http.Request, user *User, entryID int) (domainentries.ListedEntry, error) {
	entry, err := a.entryQueryService().GetEntry(r.Context(), entryID)
	if errors.Is(err, domainentries.ErrEntryNotFound) {
		return domainentries.ListedEntry{}, httpErr{status: http.StatusNotFound, detail: "Запись не найдена"}
	}
	if err != nil {
		return domainentries.ListedEntry{}, err
	}
	if !isEntryManagerRole(user.Role) && entry.OfficerID != user.ID {
		return domainentries.ListedEntry{}, httpErr{status: http.StatusForbidden, detail: "Недостаточно прав"}
	}
	if err := a.ensureOfficerInHeadScope(r.Context(), user, entry.OfficerID); err != nil {
		return domainentries.ListedEntry{}, err
	}
	return entry, nil
}

func (a *App) authorizedAttachment(r *http.Request, user *User) (domainentries.ListedEntry, domainentries.ListedAttachment, error) {
	entryID, validEntry := parseIDPathValue(r, "entry_id")
	attachmentID, validAttachment := parseIDPathValue(r, "attachment_id")
	if !validEntry || !validAttachment {
		return domainentries.ListedEntry{}, domainentries.ListedAttachment{}, httpErr{status: http.StatusNotFound, detail: "Вложение не найдено"}
	}
	entry, err := a.authorizedEntry(r, user, entryID)
	if err != nil {
		return domainentries.ListedEntry{}, domainentries.ListedAttachment{}, err
	}
	attachment, ok := findListedAttachment(entry, attachmentID)
	if !ok {
		return domainentries.ListedEntry{}, domainentries.ListedAttachment{}, httpErr{status: http.StatusNotFound, detail: "Вложение не найдено"}
	}
	return entry, attachment, nil
}

func findListedAttachment(entry domainentries.ListedEntry, attachmentID int) (domainentries.ListedAttachment, bool) {
	for _, attachment := range entry.Attachments {
		if attachment.ID == attachmentID {
			return attachment, true
		}
	}
	return domainentries.ListedAttachment{}, false
}

func (a *App) handleUpdateEntryStatus(w http.ResponseWriter, r *http.Request) {
	user, ok := requireHeadOrAdmin(a, w, r)
	if !ok {
		return
	}
	entryID, valid := parseIDPathValue(r, "entry_id")
	if !valid {
		writeError(w, http.StatusNotFound, "Запись не найдена")
		return
	}
	var payload EntryStatusUpdate
	if err := readJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Некорректные данные статуса")
		return
	}
	updated, err := a.updateKpiEntryStatus(r.Context(), user, entryID, payload)
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, listedEntryOut(updated))
}

func (a *App) handleDeleteEntry(w http.ResponseWriter, r *http.Request) {
	user, ok := requireHeadOrAdmin(a, w, r)
	if !ok {
		return
	}
	entryID, valid := parseIDPathValue(r, "entry_id")
	if !valid {
		writeError(w, http.StatusNotFound, "Запись не найдена")
		return
	}
	entry, err := a.entryQueryService().GetEntry(r.Context(), entryID)
	if errors.Is(err, domainentries.ErrEntryNotFound) {
		writeError(w, http.StatusNotFound, "Запись не найдена")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.ensureOfficerInHeadScope(r.Context(), user, entry.OfficerID); err != nil {
		writeHTTPError(w, err)
		return
	}

	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(r.Context(), `DELETE FROM direct_messages WHERE entry_id = ?`, entryID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	result, err := tx.ExecContext(r.Context(), `DELETE FROM kpi_entries WHERE id = ?`, entryID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if affected, err := result.RowsAffected(); err == nil && affected == 0 {
		writeError(w, http.StatusNotFound, "Запись не найдена")
		return
	}
	if _, err := recalculateOfficerMonth(r.Context(), tx, entry.OfficerID, entry.Month); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	storageNames := make([]string, 0, len(entry.Attachments)+1)
	storageNames = append(storageNames, entry.AttachmentStorageName)
	for _, attachment := range entry.Attachments {
		storageNames = append(storageNames, attachment.StorageName)
	}
	a.removeStoredAttachments(storageNames)
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) updateKpiEntryStatus(ctx context.Context, user *User, entryID int, payload EntryStatusUpdate) (domainentries.ListedEntry, error) {
	entry, err := a.entryQueryService().GetEntry(ctx, entryID)
	if err != nil {
		return domainentries.ListedEntry{}, entryStatusHTTPError(err)
	}
	if err := a.ensureOfficerInHeadScope(ctx, user, entry.OfficerID); err != nil {
		return domainentries.ListedEntry{}, err
	}
	service := appentries.NewStatusService(a.entryRepository(), domainentries.NewStatusPolicy())
	updated, err := service.UpdateEntryStatus(ctx, entryActor(user), domainentries.StatusUpdateRequest{
		EntryID:     entryID,
		Status:      payload.Status,
		HeadComment: payload.HeadComment,
	})
	if err != nil {
		return domainentries.ListedEntry{}, entryStatusHTTPError(err)
	}
	return a.entryQueryService().GetEntry(ctx, updated.ID)
}

func entryStatusHTTPError(err error) error {
	switch {
	case errors.Is(err, domainentries.ErrEntryNotFound):
		return httpErr{status: http.StatusNotFound, detail: "Запись не найдена"}
	case errors.Is(err, domainentries.ErrForbidden):
		return httpErr{status: http.StatusForbidden, detail: "Недостаточно прав"}
	case errors.Is(err, domainentries.ErrInvalidStatus):
		return httpErr{status: http.StatusBadRequest, detail: "Недопустимый статус"}
	default:
		return err
	}
}

func (a *App) createKpiEntry(ctx context.Context, user *User, payload EntryCreate, attachments []domainentries.Attachment) (domainentries.ListedEntry, error) {
	if isEntryManagerRole(user.Role) && payload.OfficerID != nil {
		if err := a.ensureOfficerInHeadScope(ctx, user, *payload.OfficerID); err != nil {
			return domainentries.ListedEntry{}, err
		}
	}
	service := appentries.NewService(a.entryRepository(), domainentries.NewCreationPolicy(), todayDate)
	created, err := service.CreateEntry(ctx, entryActor(user), domainentries.CreateRequest{
		CatalogID:   payload.CatalogID,
		OfficerID:   payload.OfficerID,
		Quantity:    payload.Quantity,
		Date:        payload.Date,
		Comments:    payload.Comments,
		Attachments: attachments,
	})
	if err != nil {
		return domainentries.ListedEntry{}, entryCreateHTTPError(err)
	}
	return a.entryQueryService().GetEntry(ctx, created.ID)
}

func entryActor(user *User) domainentries.Actor {
	return domainentries.Actor{
		ID:       user.ID,
		FullName: user.FullName,
		Role:     user.Role,
	}
}

func entryCreateHTTPError(err error) error {
	switch {
	case errors.Is(err, domainentries.ErrOfficerRequired):
		return httpErr{status: http.StatusBadRequest, detail: "Сотрудник не указан"}
	case errors.Is(err, domainentries.ErrOfficerNotFound):
		return httpErr{status: http.StatusNotFound, detail: "Сотрудник не найден"}
	case errors.Is(err, domainentries.ErrCatalogNotFound):
		return httpErr{status: http.StatusNotFound, detail: "Показатель KPI не найден"}
	case errors.Is(err, domainentries.ErrForbidden):
		return httpErr{status: http.StatusForbidden, detail: "Недостаточно прав"}
	case errors.Is(err, domainentries.ErrNegativeKPIForbidden):
		return httpErr{status: http.StatusForbidden, detail: "KPI с отрицательным баллом доступен только начальнику отдела"}
	case errors.Is(err, domainentries.ErrQuantityPositive):
		return httpErr{status: http.StatusBadRequest, detail: "Количество должно быть больше нуля"}
	case errors.Is(err, domainentries.ErrInvalidDate):
		return httpErr{status: http.StatusBadRequest, detail: "Дата указана неверно"}
	default:
		return err
	}
}

type uploadedAttachment struct {
	Name        string
	StorageName string
	ContentType string
	Size        int64
}

func domainAttachments(attachments []uploadedAttachment) []domainentries.Attachment {
	result := make([]domainentries.Attachment, 0, len(attachments))
	for _, attachment := range attachments {
		result = append(result, domainentries.Attachment{
			Name: attachment.Name, StorageName: attachment.StorageName,
			ContentType: attachment.ContentType, Size: attachment.Size,
		})
	}
	return result
}

func (a *App) storeAttachments(r *http.Request) ([]uploadedAttachment, error) {
	headers := r.MultipartForm.File["attachments"]
	if len(headers) == 0 {
		headers = r.MultipartForm.File["attachment"]
	}
	if len(headers) == 0 {
		return nil, httpErr{status: http.StatusBadRequest, detail: "Файл не прикреплён"}
	}
	if len(headers) > maxAttachmentsPerRequest {
		return nil, httpErr{status: http.StatusBadRequest, detail: "Слишком много вложений"}
	}
	stored := make([]uploadedAttachment, 0, len(headers))
	for _, header := range headers {
		attachment, err := a.storeUploadedAttachment(header)
		if err != nil {
			a.removeUploadedAttachments(stored)
			return nil, err
		}
		stored = append(stored, attachment)
	}
	return stored, nil
}

func (a *App) storeUploadedAttachment(header *multipart.FileHeader) (uploadedAttachment, error) {
	file, err := header.Open()
	if err != nil {
		return uploadedAttachment{}, err
	}
	defer file.Close()
	rawName := strings.ReplaceAll(strings.TrimSpace(header.Filename), "\\", "/")
	originalName := filepath.Base(rawName)
	if len(originalName) > 255 {
		originalName = originalName[:255]
	}
	if originalName == "" || originalName == "." {
		return uploadedAttachment{}, httpErr{status: http.StatusBadRequest, detail: "Файл не прикреплён"}
	}
	extension := strings.ToLower(filepath.Ext(originalName))
	if !allowedAttachmentExtensions[extension] {
		return uploadedAttachment{}, httpErr{status: http.StatusBadRequest, detail: "Недопустимый тип файла"}
	}
	if err := os.MkdirAll(a.uploadDir, 0o755); err != nil {
		return uploadedAttachment{}, err
	}
	storageName, err := randomHex(16)
	if err != nil {
		return uploadedAttachment{}, err
	}
	storageName += extension
	storagePath := filepath.Join(a.uploadDir, storageName)
	destination, err := os.Create(storagePath)
	if err != nil {
		return uploadedAttachment{}, err
	}
	size, copyErr := io.Copy(destination, io.LimitReader(file, maxAttachmentSize+1))
	closeErr := destination.Close()
	if copyErr != nil {
		_ = os.Remove(storagePath)
		return uploadedAttachment{}, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(storagePath)
		return uploadedAttachment{}, closeErr
	}
	if size > maxAttachmentSize {
		_ = os.Remove(storagePath)
		return uploadedAttachment{}, httpErr{status: http.StatusRequestEntityTooLarge, detail: "Файл слишком большой"}
	}
	contentType := normalizedAttachmentContentType(originalName, header.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if len(contentType) > 255 {
		contentType = contentType[:255]
	}
	return uploadedAttachment{Name: originalName, StorageName: storageName, ContentType: contentType, Size: size}, nil
}

func normalizedAttachmentContentType(name string, contentType string) string {
	extension := strings.ToLower(filepath.Ext(name))
	if expected := attachmentContentTypes[extension]; expected != "" {
		return expected
	}
	contentType, _, err := mime.ParseMediaType(contentType)
	if err != nil || contentType == "" {
		return "application/octet-stream"
	}
	return contentType
}

func (a *App) insertEntryAttachments(ctx context.Context, entryID int, attachments []uploadedAttachment, populateLegacy bool) error {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, attachment := range attachments {
		if _, err := tx.ExecContext(ctx, `INSERT INTO entry_attachments
			(entry_id, name, storage_name, content_type, size, created_at)
			VALUES (?, ?, ?, ?, ?, ?)`, entryID, attachment.Name, attachment.StorageName,
			attachment.ContentType, attachment.Size, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			return err
		}
	}
	if populateLegacy && len(attachments) > 0 {
		first := attachments[0]
		if _, err := tx.ExecContext(ctx, `UPDATE kpi_entries SET
			attachment_name = ?, attachment_storage_name = ?, attachment_content_type = ?, attachment_size = ?
			WHERE id = ? AND attachment_storage_name = ''`, first.Name, first.StorageName, first.ContentType, first.Size, entryID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (a *App) removeUploadedAttachments(attachments []uploadedAttachment) {
	for _, attachment := range attachments {
		_ = os.Remove(filepath.Join(a.uploadDir, attachment.StorageName))
	}
}

func (a *App) attachmentStoragePath(storageName string) (string, bool) {
	if storageName == "" || strings.ContainsAny(storageName, `/\`) || filepath.Base(storageName) != storageName {
		return "", false
	}
	storagePath := filepath.Join(a.uploadDir, storageName)
	if _, err := os.Stat(storagePath); err != nil {
		return "", false
	}
	return storagePath, true
}

func (a *App) serveStoredAttachment(w http.ResponseWriter, r *http.Request, storagePath, name, contentType string, download bool) {
	file, err := os.Open(storagePath)
	if err != nil {
		writeError(w, http.StatusNotFound, "Вложение не найдено")
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		writeError(w, http.StatusNotFound, "Вложение не найдено")
		return
	}
	disposition := "inline"
	if download {
		disposition = "attachment"
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": name}))
	w.Header().Set("Content-Type", normalizedAttachmentContentType(name, contentType))
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeContent(w, r, name, info.ModTime(), file)
}

func (a *App) createAttachmentPlaybackToken(entryID, attachmentID int) string {
	payload := fmt.Sprintf("%d:%d:%d", entryID, attachmentID, time.Now().UTC().Add(attachmentPlaybackTTL).Unix())
	signature := base64.RawURLEncoding.EncodeToString(signHS256([]byte(payload), a.secret))
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + signature
}

func (a *App) verifyAttachmentPlaybackToken(token string, entryID, attachmentID int) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return false
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	expected := base64.RawURLEncoding.EncodeToString(signHS256(payloadBytes, a.secret))
	if !hmac.Equal([]byte(parts[1]), []byte(expected)) {
		return false
	}
	values := strings.Split(string(payloadBytes), ":")
	if len(values) != 3 || values[0] != strconv.Itoa(entryID) || values[1] != strconv.Itoa(attachmentID) {
		return false
	}
	expires, err := strconv.ParseInt(values[2], 10, 64)
	return err == nil && expires >= time.Now().UTC().Unix()
}

func randomHex(size int) (string, error) {
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func quarterMonths(value string) ([3]string, error) {
	var result [3]string
	parts := strings.Split(value, "-Q")
	if len(parts) != 2 {
		return result, errors.New("invalid quarter")
	}
	year, err := parseInt(parts[0])
	if err != nil {
		return result, err
	}
	quarter, err := parseInt(parts[1])
	if err != nil || quarter < 1 || quarter > 4 {
		return result, errors.New("invalid quarter")
	}
	first := (quarter-1)*3 + 1
	for i := 0; i < 3; i++ {
		result[i] = fmt.Sprintf("%04d-%02d", year, first+i)
	}
	return result, nil
}

type httpErr struct {
	status int
	detail string
}

func (e httpErr) Error() string {
	return e.detail
}

func writeHTTPError(w http.ResponseWriter, err error) {
	var hErr httpErr
	if errors.As(err, &hErr) {
		writeError(w, hErr.status, hErr.detail)
		return
	}
	writeError(w, http.StatusInternalServerError, err.Error())
}
