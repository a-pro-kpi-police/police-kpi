package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLeadershipAccountLifecycleAndLogin(t *testing.T) {
	a := newSupportTestApp(t)
	defer a.db.Close()
	h := a.routes()
	admin, err := a.getUserByUsername(t.Context(), "admin")
	if err != nil {
		t.Fatal(err)
	}
	adminToken := supportToken(t, a, admin)
	district := insertSupportDistrict(t, a)
	payload := map[string]any{"username": "leadership", "password": "pass123", "full_name": "Руководство", "role": roleLeadership, "district_id": district}
	r := supportJSONRequest(t, h, "POST", "/api/admin/users", adminToken, payload)
	if r.Code != 200 {
		t.Fatalf("create: %d %s", r.Code, r.Body)
	}
	var created AdminUserOut
	decodeSupportResponse(t, r, &created)
	if created.Role != roleLeadership || created.District == nil || created.District.ID != district {
		t.Fatalf("created: %+v", created)
	}
	login := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(url.Values{"username": {"leadership"}, "password": {"pass123"}}.Encode()))
	login.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	h.ServeHTTP(response, login)
	if response.Code != 200 {
		t.Fatalf("login: %d %s", response.Code, response.Body)
	}
	var token TokenOut
	decodeSupportResponse(t, response, &token)
	r = supportJSONRequest(t, h, "GET", "/api/auth/me", token.AccessToken, nil)
	if r.Code != 200 || !strings.Contains(r.Body.String(), `"role":"leadership"`) {
		t.Fatalf("profile: %d %s", r.Code, r.Body)
	}
	path := "/api/admin/users/" + intString(created.ID)
	payload["district_id"] = nil
	payload["full_name"] = "Обновлённое руководство"
	payload["password"] = "new-password"
	r = supportJSONRequest(t, h, "PATCH", path, adminToken, payload)
	if r.Code != 200 {
		t.Fatalf("update: %d %s", r.Code, r.Body)
	}
	var updated AdminUserOut
	decodeSupportResponse(t, r, &updated)
	if updated.District != nil || updated.FullName != payload["full_name"] {
		t.Fatalf("updated: %+v", updated)
	}
	user, err := a.getUserByID(t.Context(), created.ID)
	if err != nil || !verifyPassword("new-password", user.PasswordHash) {
		t.Fatalf("password not updated: %v", err)
	}
	r = supportJSONRequest(t, h, "GET", "/api/admin/users", adminToken, nil)
	if r.Code != 200 || !strings.Contains(r.Body.String(), `"role":"leadership"`) {
		t.Fatalf("list: %d %s", r.Code, r.Body)
	}
	// Updating without a password preserves it; existing sessions use the new scope.
	payload["district_id"] = district
	payload["password"] = ""
	r = supportJSONRequest(t, h, "PATCH", path, adminToken, payload)
	if r.Code != 200 {
		t.Fatalf("update scope without password: %d %s", r.Code, r.Body)
	}
	user, err = a.getUserByID(t.Context(), created.ID)
	if err != nil || !verifyPassword("new-password", user.PasswordHash) {
		t.Fatalf("password changed on empty password update: %v", err)
	}
	r = supportJSONRequest(t, h, "GET", "/api/districts", token.AccessToken, nil)
	var scopedDistricts []District
	decodeSupportResponse(t, r, &scopedDistricts)
	if r.Code != 200 || len(scopedDistricts) != 1 || scopedDistricts[0].ID != district {
		t.Fatalf("existing session scope: %d %s", r.Code, r.Body)
	}
	payload["district_id"] = district + 10000
	r = supportJSONRequest(t, h, "PATCH", path, adminToken, payload)
	if r.Code != 404 {
		t.Fatalf("invalid district: %d %s", r.Code, r.Body)
	}
	payload["district_id"] = district
	// Creating or promoting an administrator remains forbidden.
	payload["role"] = roleAdmin
	for _, method := range []string{"POST", "PATCH"} {
		target := path
		if method == "POST" {
			target = "/api/admin/users"
		}
		r = supportJSONRequest(t, h, method, target, adminToken, payload)
		if r.Code != 400 {
			t.Fatalf("admin promotion %s: %d %s", method, r.Code, r.Body)
		}
	}
	r = supportJSONRequest(t, h, "DELETE", path, adminToken, nil)
	if r.Code != 204 {
		t.Fatalf("delete: %d %s", r.Code, r.Body)
	}
	r = supportJSONRequest(t, h, "GET", "/api/auth/me", token.AccessToken, nil)
	if r.Code != 401 {
		t.Fatalf("deleted session: %d", r.Code)
	}
}

func TestLeadershipScopeAndReadOnlyAPI(t *testing.T) {
	a := newSupportTestApp(t)
	defer a.db.Close()
	h := a.routes()
	d1 := insertSupportDistrict(t, a)
	var d2 int
	if err := a.db.QueryRowContext(t.Context(), `INSERT INTO districts(name, coeff) VALUES ('Other District', 1) RETURNING id`).Scan(&d2); err != nil {
		t.Fatal(err)
	}
	inside := insertSupportUser(t, a, "inside", "Inside", "", roleOfficer, &d1)
	outside := insertSupportUser(t, a, "outside", "Outside", "", roleOfficer, &d2)
	local := insertSupportUser(t, a, "local-leader", "Local leader", "", roleLeadership, &d1)
	global := insertSupportUser(t, a, "global-leader", "Global leader", "", roleLeadership, nil)
	var catalog int
	if err := a.db.QueryRowContext(t.Context(), `INSERT INTO kpi_catalog(code, indicator, block, points_per_unit, is_active) VALUES ('LEADERSHIP-TEST', 'Test', 'Test', 10, 1) RETURNING id`).Scan(&catalog); err != nil {
		t.Fatal(err)
	}
	entryIDs := []int{}
	for i, officer := range []*User{inside, outside} {
		q := float64(i + 1)
		date := []string{"2026-08-01", "2026-09-01"}[i]
		entry, err := a.createKpiEntry(t.Context(), officer, EntryCreate{CatalogID: catalog, Quantity: &q, Date: &date}, nil)
		if err != nil {
			t.Fatal(err)
		}
		entryIDs = append(entryIDs, entry.ID)
		if _, err := a.db.ExecContext(t.Context(), `UPDATE kpi_entries SET status = 'confirmed' WHERE id = ?`, entry.ID); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(a.uploadDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(a.uploadDir, "leadership.mp4"), []byte("attachment"), 0600); err != nil {
		t.Fatal(err)
	}
	attachments := []int{}
	for _, entryID := range entryIDs {
		var id int
		if err := a.db.QueryRowContext(t.Context(), `INSERT INTO entry_attachments(entry_id, name, storage_name, content_type, size, created_at) VALUES (?, 'test.mp4', 'leadership.mp4', 'video/mp4', 10, '2026-08-01T00:00:00Z') RETURNING id`, entryID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		attachments = append(attachments, id)
		if _, err := a.db.ExecContext(t.Context(), `UPDATE kpi_entries SET attachment_name='test.mp4', attachment_storage_name='leadership.mp4' WHERE id=?`, entryID); err != nil {
			t.Fatal(err)
		}
	}
	localToken, globalToken := supportToken(t, a, local), supportToken(t, a, global)
	for _, path := range []string{"/api/catalog", "/api/districts", "/api/officers", "/api/rating", "/api/months", "/api/entries", "/api/dashboard", "/api/target-overview?month=2026-08", "/api/targets?month=2026-08&officer_id=" + intString(inside.ID), "/api/target-progress?officer_id=" + intString(inside.ID), "/api/group-targets?month=2026-08&district_id=" + intString(d1)} {
		for _, token := range []string{localToken, globalToken} {
			r := supportJSONRequest(t, h, "GET", path, token, nil)
			if r.Code != 200 {
				t.Fatalf("view %s: %d %s", path, r.Code, r.Body)
			}
			if token == localToken && (strings.Contains(r.Body.String(), `Outside`) || strings.Contains(r.Body.String(), `2026-09`)) {
				t.Fatalf("scope leak %s: %s", path, r.Body)
			}
		}
	}
	for _, test := range []struct {
		path                    string
		localCount, globalCount int
	}{
		{"/api/officers", 1, 2}, {"/api/rating", 1, 2}, {"/api/entries", 1, 2}, {"/api/districts", 1, 2},
	} {
		for _, tc := range []struct {
			token string
			count int
		}{{localToken, test.localCount}, {globalToken, test.globalCount}} {
			r := supportJSONRequest(t, h, "GET", test.path, tc.token, nil)
			var rows []json.RawMessage
			decodeSupportResponse(t, r, &rows)
			if len(rows) != tc.count {
				t.Fatalf("%s: got %d rows, want %d: %s", test.path, len(rows), tc.count, r.Body)
			}
		}
	}
	for _, path := range []string{
		"/api/dashboard?district_id=" + intString(d2), "/api/dashboard?officer_id=" + intString(outside.ID),
		"/api/entries?officer_id=" + intString(outside.ID), "/api/target-overview?officer_id=" + intString(outside.ID),
		"/api/target-overview?district_id=" + intString(d2), "/api/group-targets?month=2026-08&district_id=" + intString(d2),
		"/api/targets?month=2026-08&officer_id=" + intString(outside.ID), "/api/target-progress?officer_id=" + intString(outside.ID),
	} {
		r := supportJSONRequest(t, h, "GET", path, localToken, nil)
		if r.Code != 403 {
			t.Fatalf("foreign %s: %d %s", path, r.Code, r.Body)
		}
		r = supportJSONRequest(t, h, "GET", path, globalToken, nil)
		if r.Code != 200 {
			t.Fatalf("global %s: %d %s", path, r.Code, r.Body)
		}
	}
	for i, entry := range entryIDs {
		for _, suffix := range []string{"/attachment", "/attachments/" + intString(attachments[i]), "/attachments/" + intString(attachments[i]) + "/playback-url"} {
			path := "/api/entries/" + intString(entry) + suffix
			want := 200
			if i == 1 {
				want = 403
			}
			r := supportJSONRequest(t, h, "GET", path, localToken, nil)
			if r.Code != want {
				t.Fatalf("attachment %s: %d %s", path, r.Code, r.Body)
			}
			r = supportJSONRequest(t, h, "GET", path, globalToken, nil)
			if r.Code != 200 {
				t.Fatalf("global attachment %s: %d %s", path, r.Code, r.Body)
			}
		}
	}
	// The average and comparison denominator cannot include the other district.
	r := supportJSONRequest(t, h, "GET", "/api/dashboard", localToken, nil)
	var report struct {
		Average float64 `json:"global_average_score"`
		Count   int     `json:"officers_count"`
	}
	decodeSupportResponse(t, r, &report)
	if report.Average != 10 || report.Count != 1 {
		t.Fatalf("scoped summary: %+v", report)
	}
	// Invalid/foreign filter values never expand collection access.
	r = supportJSONRequest(t, h, "GET", "/api/entries?district_id="+intString(d2)+"&limit=100&offset=0", localToken, nil)
	if r.Code != 200 || strings.Contains(r.Body.String(), "Outside") {
		t.Fatalf("tampered filter: %d %s", r.Code, r.Body)
	}
	// Snapshot the entire database to verify all denied requests have no effects.
	snapshot := func() string {
		var dump strings.Builder
		for _, table := range []string{"users", "kpi_entries", "kpi_catalog", "kpi_targets", "direct_messages", "support_conversations", "support_messages", "entry_attachments"} {
			rows, err := a.db.QueryContext(context.Background(), "SELECT * FROM "+table+" ORDER BY id")
			if err != nil {
				t.Fatal(err)
			}
			cols, _ := rows.Columns()
			for rows.Next() {
				values := make([]any, len(cols))
				pointers := make([]any, len(cols))
				for i := range values {
					pointers[i] = &values[i]
				}
				if err := rows.Scan(pointers...); err != nil {
					t.Fatal(err)
				}
				b, err := json.Marshal(values)
				if err != nil {
					t.Fatal(err)
				}
				dump.Write(b)
			}
			if err := rows.Err(); err != nil {
				t.Fatal(err)
			}
			rows.Close()
		}
		return dump.String()
	}
	before := snapshot()
	for _, token := range []string{localToken, globalToken} {
		for _, tc := range []struct{ method, path string }{
			{"POST", "/api/entries"}, {"POST", "/api/entries/with-attachment"}, {"POST", "/api/entries/with-attachments"},
			{"POST", "/api/entries/" + intString(entryIDs[0]) + "/attachments"}, {"PATCH", "/api/entries/" + intString(entryIDs[0]) + "/status"}, {"DELETE", "/api/entries/" + intString(entryIDs[0])},
			{"PUT", "/api/targets"}, {"PUT", "/api/group-targets"},
			{"GET", "/api/admin/users"}, {"POST", "/api/admin/users"}, {"PATCH", "/api/admin/users/" + intString(inside.ID)}, {"DELETE", "/api/admin/users/" + intString(inside.ID)},
			{"GET", "/api/admin/catalog"}, {"POST", "/api/admin/catalog"}, {"PATCH", "/api/admin/catalog/" + intString(catalog)}, {"DELETE", "/api/admin/catalog/" + intString(catalog)},
			{"GET", "/api/admin/rating"}, {"GET", "/api/admin/officers/" + intString(inside.ID) + "/dashboard"}, {"GET", "/api/admin/officers/" + intString(inside.ID) + "/target-overview"},
			{"GET", "/api/messenger/contacts"}, {"GET", "/api/messenger/unread-count"}, {"GET", "/api/messenger/entries/1"}, {"GET", "/api/messenger/messages/1"}, {"POST", "/api/messenger/messages"}, {"DELETE", "/api/messenger/messages/1"},
			{"GET", "/api/support/conversation"}, {"POST", "/api/support/messages"}, {"GET", "/api/admin/support/conversations"}, {"GET", "/api/admin/support/conversations/1/messages"}, {"POST", "/api/admin/support/conversations/1/messages"},
		} {
			r := supportJSONRequest(t, h, tc.method, tc.path, token, map[string]any{"officer_id": inside.ID, "catalog_id": catalog, "quantity": 1, "status": "rejected", "district_id": d1})
			if r.Code != http.StatusForbidden {
				t.Fatalf("write/admin %s %s: %d %s", tc.method, tc.path, r.Code, r.Body)
			}
		}
	}
	if after := snapshot(); after != before {
		t.Fatal("denied requests changed persisted data")
	}
	// Existing roles keep their scopes and moderation/creation rights.
	head := insertSupportUser(t, a, "head-regression", "Head", "", roleHead, &d1)
	admin, err := a.getUserByUsername(t.Context(), "admin")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		user                        *User
		listCount, moderationStatus int
	}{
		{inside, 1, 403}, {head, 1, 200}, {admin, 2, 200},
	} {
		token := supportToken(t, a, tc.user)
		r := supportJSONRequest(t, h, "GET", "/api/entries", token, nil)
		var entries []EntryOut
		decodeSupportResponse(t, r, &entries)
		if r.Code != 200 || len(entries) != tc.listCount {
			t.Fatalf("%s entry scope: %d %s", tc.user.Role, r.Code, r.Body)
		}
		r = supportJSONRequest(t, h, "PATCH", "/api/entries/"+intString(entryIDs[0])+"/status", token, map[string]any{"status": "confirmed"})
		if r.Code != tc.moderationStatus {
			t.Fatalf("%s moderation: %d %s", tc.user.Role, r.Code, r.Body)
		}
		payload := map[string]any{"catalog_id": catalog, "quantity": 1, "officer_id": inside.ID}
		r = supportJSONRequest(t, h, "POST", "/api/entries", token, payload)
		if r.Code != 200 {
			t.Fatalf("%s creation: %d %s", tc.user.Role, r.Code, r.Body)
		}
		if _, err := a.db.ExecContext(t.Context(), `DELETE FROM kpi_entries WHERE id NOT IN (?, ?)`, entryIDs[0], entryIDs[1]); err != nil {
			t.Fatal(err)
		}
	}
}
