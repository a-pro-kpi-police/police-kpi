package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestStoreAttachmentsAcceptsMultipleSupportedVideos(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for _, item := range []struct {
		name string
		data string
	}{{"first.mp4", "mp4-data"}, {"second.webm", "webm-data"}} {
		part, err := writer.CreateFormFile("attachments", item.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write([]byte(item.data)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/entries/with-attachments", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if err := request.ParseMultipartForm(1024); err != nil {
		t.Fatal(err)
	}
	defer request.MultipartForm.RemoveAll()
	app := &App{uploadDir: t.TempDir()}
	attachments, err := app.storeAttachments(request)
	if err != nil {
		t.Fatal(err)
	}
	defer app.removeUploadedAttachments(attachments)
	if len(attachments) != 2 {
		t.Fatalf("stored %d attachments", len(attachments))
	}
	if attachments[0].ContentType != "video/mp4" || attachments[1].ContentType != "video/webm" {
		t.Fatalf("content types = %q, %q", attachments[0].ContentType, attachments[1].ContentType)
	}
}

func TestVideoAttachmentPlaybackSupportsRangesAndPermissions(t *testing.T) {
	t.Setenv("KPI_DATABASE_URL", "sqlite:///:memory:")
	t.Setenv("KPI_UPLOAD_DIR", filepath.Join(t.TempDir(), "uploads"))
	t.Setenv("KPI_SECRET", "attachment-test-secret")
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()

	passwordHash, err := hashPassword("test-password")
	if err != nil {
		t.Fatal(err)
	}
	districtResult, err := app.db.ExecContext(t.Context(), `INSERT INTO districts (name, coeff) VALUES (?, ?)`, "Attachment test", 1.0)
	if err != nil {
		t.Fatal(err)
	}
	districtID, err := districtResult.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	insertOfficer := func(username string) int {
		result, err := app.db.ExecContext(t.Context(), `INSERT INTO users
			(username, password_hash, full_name, rank, role, district_id)
			VALUES (?, ?, ?, ?, ?, ?)`, username, passwordHash, username, "", roleOfficer, districtID)
		if err != nil {
			t.Fatal(err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}
		return int(id)
	}
	ownerID := insertOfficer("video-owner")
	otherOfficerID := insertOfficer("video-outsider")
	catalogResult, err := app.db.ExecContext(t.Context(), `INSERT INTO kpi_catalog
		(code, indicator, block, points_per_unit, is_active) VALUES (?, ?, ?, ?, 1)`, "VIDEO-TEST", "Video", "Test", 1.0)
	if err != nil {
		t.Fatal(err)
	}
	catalogIDValue, err := catalogResult.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	catalogID := int(catalogIDValue)
	owner, err := app.getUserByID(t.Context(), ownerID)
	if err != nil {
		t.Fatal(err)
	}
	quantity := 1.0
	entry, err := app.createKpiEntry(t.Context(), owner, EntryCreate{CatalogID: catalogID, Quantity: &quantity}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(app.uploadDir, 0o755); err != nil {
		t.Fatal(err)
	}
	const storageName = "test-video.mp4"
	if err := os.WriteFile(filepath.Join(app.uploadDir, storageName), []byte("0123456789"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := app.db.ExecContext(t.Context(), `INSERT INTO entry_attachments
		(entry_id, name, storage_name, content_type, size, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`, entry.ID, "evidence.mp4", storageName, "application/octet-stream", 10, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		t.Fatal(err)
	}
	attachmentID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}

	ownerToken, err := app.createToken(owner)
	if err != nil {
		t.Fatal(err)
	}
	playbackURLPath := "/api/entries/" + strconv.Itoa(entry.ID) + "/attachments/" + strconv.Itoa(int(attachmentID)) + "/playback-url"
	request := httptest.NewRequest(http.MethodGet, playbackURLPath, nil)
	request.Header.Set("Authorization", "Bearer "+ownerToken)
	response := httptest.NewRecorder()
	app.routes().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("playback URL status = %d, body = %s", response.Code, response.Body.String())
	}
	var playback AttachmentPlaybackOut
	if err := json.NewDecoder(response.Body).Decode(&playback); err != nil {
		t.Fatal(err)
	}

	request = httptest.NewRequest(http.MethodGet, playback.URL, nil)
	request.Header.Set("Range", "bytes=2-5")
	response = httptest.NewRecorder()
	app.routes().ServeHTTP(response, request)
	if response.Code != http.StatusPartialContent {
		t.Fatalf("range status = %d, body = %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != "video/mp4" {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := response.Header().Get("Accept-Ranges"); got != "bytes" {
		t.Fatalf("Accept-Ranges = %q", got)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "2345" {
		t.Fatalf("range body = %q", body)
	}

	otherOfficer, err := app.getUserByID(t.Context(), otherOfficerID)
	if err != nil {
		t.Fatal(err)
	}
	otherToken, err := app.createToken(otherOfficer)
	if err != nil {
		t.Fatal(err)
	}
	request = httptest.NewRequest(http.MethodGet, playbackURLPath, strings.NewReader(""))
	request.Header.Set("Authorization", "Bearer "+otherToken)
	response = httptest.NewRecorder()
	app.routes().ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("other officer status = %d, body = %s", response.Code, response.Body.String())
	}
}
