package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	appentries "kpi-police/backend-golang/internal/application/entries"
	domainentries "kpi-police/backend-golang/internal/domain/entries"
)

const maxAttachmentSize = 20 * 1024 * 1024

var allowedAttachmentExtensions = map[string]bool{
	".csv": true, ".doc": true, ".docx": true, ".jpeg": true, ".jpg": true,
	".pdf": true, ".png": true, ".txt": true, ".webp": true, ".xls": true,
	".xlsx": true, ".zip": true,
}

func (a *App) handleListEntries(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(a, w, r)
	if !ok {
		return
	}
	queryValues := r.URL.Query()
	entryQuery := domainentries.EntryQuery{Actor: entryActor(user)}
	if user.Role == roleHead {
		if officerText := queryValues.Get("officer_id"); officerText != "" {
			officerID, err := parseInt(officerText)
			if err != nil {
				writeError(w, http.StatusBadRequest, "Сотрудник указан неверно")
				return
			}
			entryQuery.OfficerID = &officerID
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
	created, err := a.createKpiEntry(r.Context(), user, payload, domainentries.Attachment{})
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
	r.Body = http.MaxBytesReader(w, r.Body, maxAttachmentSize+2*1024*1024)
	if err := r.ParseMultipartForm(maxAttachmentSize + 1024*1024); err != nil {
		writeError(w, http.StatusBadRequest, "Некорректная форма вложения")
		return
	}
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
	originalName, storageName, contentType, size, err := a.storeAttachment(r)
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	created, err := a.createKpiEntry(r.Context(), user, payload, domainentries.Attachment{
		Name:        originalName,
		StorageName: storageName,
		ContentType: contentType,
		Size:        size,
	})
	if err != nil {
		_ = os.Remove(filepath.Join(a.uploadDir, storageName))
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
	if user.Role != roleHead && entry.OfficerID != user.ID {
		writeError(w, http.StatusForbidden, "Недостаточно прав")
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
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": entry.AttachmentName}))
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, storagePath)
}

func (a *App) handleUpdateEntryStatus(w http.ResponseWriter, r *http.Request) {
	user, ok := requireHead(a, w, r)
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

func (a *App) updateKpiEntryStatus(ctx context.Context, user *User, entryID int, payload EntryStatusUpdate) (domainentries.ListedEntry, error) {
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

func (a *App) createKpiEntry(ctx context.Context, user *User, payload EntryCreate, attachment domainentries.Attachment) (domainentries.ListedEntry, error) {
	service := appentries.NewService(a.entryRepository(), domainentries.NewCreationPolicy(), todayDate)
	created, err := service.CreateEntry(ctx, entryActor(user), domainentries.CreateRequest{
		CatalogID:  payload.CatalogID,
		OfficerID:  payload.OfficerID,
		Quantity:   payload.Quantity,
		Date:       payload.Date,
		Comments:   payload.Comments,
		Attachment: attachment,
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

func (a *App) storeAttachment(r *http.Request) (string, string, string, int64, error) {
	file, header, err := r.FormFile("attachment")
	if err != nil {
		return "", "", "", 0, httpErr{status: http.StatusBadRequest, detail: "Файл не прикреплён"}
	}
	defer file.Close()
	rawName := strings.ReplaceAll(strings.TrimSpace(header.Filename), "\\", "/")
	originalName := filepath.Base(rawName)
	if len(originalName) > 255 {
		originalName = originalName[:255]
	}
	if originalName == "" || originalName == "." {
		return "", "", "", 0, httpErr{status: http.StatusBadRequest, detail: "Файл не прикреплён"}
	}
	extension := strings.ToLower(filepath.Ext(originalName))
	if !allowedAttachmentExtensions[extension] {
		return "", "", "", 0, httpErr{status: http.StatusBadRequest, detail: "Недопустимый тип файла"}
	}
	if err := os.MkdirAll(a.uploadDir, 0o755); err != nil {
		return "", "", "", 0, err
	}
	storageName, err := randomHex(16)
	if err != nil {
		return "", "", "", 0, err
	}
	storageName += extension
	storagePath := filepath.Join(a.uploadDir, storageName)
	destination, err := os.Create(storagePath)
	if err != nil {
		return "", "", "", 0, err
	}
	size, copyErr := io.Copy(destination, io.LimitReader(file, maxAttachmentSize+1))
	closeErr := destination.Close()
	if copyErr != nil {
		_ = os.Remove(storagePath)
		return "", "", "", 0, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(storagePath)
		return "", "", "", 0, closeErr
	}
	if size > maxAttachmentSize {
		_ = os.Remove(storagePath)
		return "", "", "", 0, httpErr{status: http.StatusRequestEntityTooLarge, detail: "Файл слишком большой"}
	}
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if len(contentType) > 255 {
		contentType = contentType[:255]
	}
	return originalName, storageName, contentType, size, nil
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
