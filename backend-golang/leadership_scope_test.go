package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLeadershipScopeMigration(t *testing.T) {
	a := newSupportTestApp(t)
	defer a.db.Close()
	d1 := insertSupportDistrict(t, a)
	local := insertSupportUser(t, a, "legacy-local", "Local", "", roleLeadership, &d1)
	global := insertSupportUser(t, a, "legacy-all", "All", "", roleLeadership, nil)
	head := insertSupportUser(t, a, "legacy-head", "Head", "", roleHead, &d1)
	officer := insertSupportUser(t, a, "legacy-officer", "Officer", "", roleOfficer, &d1)
	// Exercise startup against an actual pre-#29 schema, including column addition.
	if _, err := a.db.ExecContext(t.Context(), `ALTER TABLE users DROP COLUMN leadership_scope`); err != nil {
		t.Fatal(err)
	}
	if err := createSchema(a.db); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		id   int
		mode string
		ids  []int
	}{
		{local.ID, "selected", []int{d1}}, {global.ID, "all", []int{}},
	} {
		user, err := a.getUserByID(t.Context(), tc.id)
		if err != nil {
			t.Fatal(err)
		}
		if user.LeadershipScope.Mode != tc.mode || !reflect.DeepEqual(user.LeadershipScope.DistrictIDs, tc.ids) {
			t.Fatalf("migration: %+v", user.LeadershipScope)
		}
	}
	// Affiliation updates and subsequent restarts must not grant additional access.
	if _, err := a.db.ExecContext(t.Context(), `UPDATE users SET district_id = NULL WHERE id = ?`, local.ID); err != nil {
		t.Fatal(err)
	}
	if err := createSchema(a.db); err != nil {
		t.Fatal(err)
	}
	user, err := a.getUserByID(t.Context(), local.ID)
	if err != nil || !reflect.DeepEqual(user.LeadershipScope.DistrictIDs, []int{d1}) || user.LeadershipScope.Mode != "selected" {
		t.Fatalf("restart expanded scope: %+v %v", user, err)
	}
	for _, u := range []*User{head, officer} {
		var role, raw string
		var district int
		if err := a.db.QueryRowContext(t.Context(), `SELECT role,district_id,leadership_scope FROM users WHERE id=?`, u.ID).Scan(&role, &district, &raw); err != nil {
			t.Fatal(err)
		}
		if role != u.Role || district != d1 || raw != "" {
			t.Fatalf("other role changed: %s %d %s", role, district, raw)
		}
	}
}

func TestLeadershipMultipleCompaniesAndLiveAccountUpdates(t *testing.T) {
	a := newSupportTestApp(t)
	defer a.db.Close()
	h := a.routes()
	admin, err := a.getUserByUsername(t.Context(), "admin")
	if err != nil {
		t.Fatal(err)
	}
	adminToken := supportToken(t, a, admin)
	districts := []int{}
	officers := []*User{}
	entries := []int{}
	var catalog int
	if err := a.db.QueryRowContext(t.Context(), `INSERT INTO kpi_catalog(code,indicator,block,points_per_unit,is_active) VALUES('MULTI','Multi','Test',10,1) RETURNING id`).Scan(&catalog); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 3; i++ {
		var id int
		if err := a.db.QueryRowContext(t.Context(), `INSERT INTO districts(name,coeff) VALUES(?,1) RETURNING id`, fmt.Sprintf("Company %d", i)).Scan(&id); err != nil {
			t.Fatal(err)
		}
		districts = append(districts, id)
		officer := insertSupportUser(t, a, fmt.Sprintf("multi-%d", i), fmt.Sprintf("Officer%d", i), "", roleOfficer, &id)
		officers = append(officers, officer)
		date := fmt.Sprintf("2026-%02d-01", i)
		quantity := float64(i)
		entry, err := a.createKpiEntry(t.Context(), officer, EntryCreate{CatalogID: catalog, Quantity: &quantity, Date: &date}, nil)
		if err != nil {
			t.Fatal(err)
		}
		entries = append(entries, entry.ID)
		if _, err := a.db.ExecContext(t.Context(), `UPDATE kpi_entries SET status='confirmed' WHERE id=?`, entry.ID); err != nil {
			t.Fatal(err)
		}
	}
	scope := func(mode string, ids ...int) map[string]any { return map[string]any{"mode": mode, "district_ids": ids} }
	for i, settings := range []*LeadershipScope{
		{Mode: "selected", DistrictIDs: []int{districts[0]}},
		{Mode: "selected", DistrictIDs: []int{districts[0], districts[2], districts[0]}},
		{Mode: "all", DistrictIDs: []int{}},
	} {
		payload := map[string]any{"username": fmt.Sprintf("create-mode-%d", i), "password": "pass123", "full_name": "Create mode", "role": roleLeadership, "leadership_scope": settings}
		r := supportJSONRequest(t, h, "POST", "/api/admin/users", adminToken, payload)
		if r.Code != 200 {
			t.Fatalf("create mode: %d %s", r.Code, r.Body)
		}
		var created AdminUserOut
		decodeSupportResponse(t, r, &created)
		want := []int{districts[0]}
		if i == 1 {
			want = []int{districts[0], districts[2]}
		}
		if i == 2 {
			want = []int{}
		}
		if created.LeadershipScope.Mode != settings.Mode || !reflect.DeepEqual(created.LeadershipScope.DistrictIDs, want) {
			t.Fatalf("create settings: %+v", created.LeadershipScope)
		}
	}
	// Affiliation deliberately belongs to the forbidden company.
	payload := map[string]any{"username": "multi-leader", "password": "pass123", "full_name": "Multi leader", "role": roleLeadership, "district_id": districts[1], "leadership_scope": scope("selected", districts[0], districts[2])}
	r := supportJSONRequest(t, h, "POST", "/api/admin/users", adminToken, payload)
	if r.Code != 200 {
		t.Fatalf("create multi: %d %s", r.Code, r.Body)
	}
	var account AdminUserOut
	decodeSupportResponse(t, r, &account)
	user, err := a.getUserByID(t.Context(), account.ID)
	if err != nil {
		t.Fatal(err)
	}
	token := supportToken(t, a, user)
	accountPath := "/api/admin/users/" + intString(account.ID)
	request := func(path string, want int) string {
		t.Helper()
		r := supportJSONRequest(t, h, "GET", path, token, nil)
		if r.Code != want {
			t.Fatalf("%s: %d want %d: %s", path, r.Code, want, r.Body)
		}
		return r.Body.String()
	}
	for _, path := range []string{"/api/districts", "/api/officers", "/api/rating", "/api/entries", "/api/dashboard", "/api/target-overview?month=2026-01"} {
		body := request(path, 200)
		if strings.Contains(body, "Officer2") || strings.Contains(body, "Company 2") {
			t.Fatalf("multi scope leak %s: %s", path, body)
		}
		if path != "/api/districts" && (!strings.Contains(body, "Officer1") || !strings.Contains(body, "Officer3")) {
			t.Fatalf("missing allowed officers %s: %s", path, body)
		}
	}
	for _, id := range []int{officers[0].ID, officers[2].ID} {
		for _, path := range []string{"/api/dashboard?officer_id=", "/api/entries?officer_id=", "/api/targets?month=2026-01&officer_id=", "/api/target-progress?month=2026-01&officer_id=", "/api/target-overview?month=2026-01&officer_id="} {
			request(path+intString(id), 200)
		}
	}
	for _, path := range []string{"/api/dashboard?officer_id=", "/api/entries?officer_id=", "/api/targets?month=2026-01&officer_id=", "/api/target-progress?officer_id=", "/api/target-overview?month=2026-01&officer_id="} {
		request(path+intString(officers[1].ID), 403)
	}
	for _, path := range []string{"/api/dashboard?district_id=", "/api/target-overview?month=2026-01&district_id=", "/api/group-targets?month=2026-01&district_id=", "/api/entries?district_id=", "/api/rating?district_id=", "/api/officers?district_id="} {
		request(path+intString(districts[1]), 403)
	}
	// Direct entry IDs (including a legacy attachment path) cannot bypass the boundary.
	request("/api/entries/"+intString(entries[1])+"/attachment", 403)
	var months []string
	if err := json.Unmarshal([]byte(request("/api/months", 200)), &months); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(months, []string{"2026-03", "2026-01"}) {
		t.Fatalf("months: %v", months)
	}
	var summary struct {
		Average float64 `json:"global_average_score"`
		Count   int     `json:"officers_count"`
		Total   float64 `json:"total_final"`
	}
	if err := json.Unmarshal([]byte(request("/api/dashboard", 200)), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.Average != 20 || summary.Count != 2 || summary.Total != 40 {
		t.Fatalf("multi aggregate: %+v", summary)
	}
	if err := os.MkdirAll(a.uploadDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(a.uploadDir, "multi.mp4"), []byte("test-video"), 0600); err != nil {
		t.Fatal(err)
	}
	playbackURLs := []string{}
	for _, entryID := range entries {
		var attachmentID int
		if err := a.db.QueryRowContext(t.Context(), `INSERT INTO entry_attachments(entry_id,name,storage_name,content_type,size,created_at) VALUES(?,'multi.mp4','multi.mp4','video/mp4',10,'2026-01-01T00:00:00Z') RETURNING id`, entryID).Scan(&attachmentID); err != nil {
			t.Fatal(err)
		}
		path := "/api/entries/" + intString(entryID) + "/attachments/" + intString(attachmentID)
		if entryID == entries[1] {
			request(path, 403)
			request(path+"/playback-url", 403)
			continue
		}
		request(path, 200)
		var playback AttachmentPlaybackOut
		if err := json.Unmarshal([]byte(request(path+"/playback-url", 200)), &playback); err != nil {
			t.Fatal(err)
		}
		// Video requests intentionally carry no Bearer token.
		r := supportJSONRequest(t, h, "GET", playback.URL, "", nil)
		if r.Code != 200 {
			t.Fatalf("playback: %d %s", r.Code, r.Body)
		}
		playbackURLs = append(playbackURLs, playback.URL)
	}
	// Empty/missing/invalid scopes fail on both creation and editing without changing persisted access.
	for _, bad := range []any{nil, scope("selected"), scope("selected", -1), scope("selected", 999999), scope("invalid", districts[0])} {
		payload["leadership_scope"] = bad
		for _, method := range []string{"POST", "PATCH"} {
			path := accountPath
			if method == "POST" {
				path = "/api/admin/users"
			}
			r := supportJSONRequest(t, h, method, path, adminToken, payload)
			if r.Code != 400 && r.Code != 404 {
				t.Fatalf("invalid scope %v %s: %d %s", bad, method, r.Code, r.Body)
			}
		}
		request("/api/dashboard?officer_id="+intString(officers[2].ID), 200)
	}
	// One -> all -> multiple updates take effect for the SAME token.
	for _, next := range []struct {
		mode  string
		ids   []int
		count int
	}{
		{"selected", []int{districts[0]}, 1}, {"all", []int{}, 3}, {"selected", []int{districts[0], districts[2]}, 2},
	} {
		payload["leadership_scope"] = scope(next.mode, next.ids...)
		r := supportJSONRequest(t, h, "PATCH", accountPath, adminToken, payload)
		if r.Code != 200 {
			t.Fatalf("update: %d %s", r.Code, r.Body)
		}
		var saved AdminUserOut
		decodeSupportResponse(t, r, &saved)
		if saved.LeadershipScope.Mode != next.mode || !reflect.DeepEqual(saved.LeadershipScope.DistrictIDs, next.ids) {
			t.Fatalf("saved settings: %+v", saved.LeadershipScope)
		}
		var rows []OfficerOut
		if err := json.Unmarshal([]byte(request("/api/officers", 200)), &rows); err != nil {
			t.Fatal(err)
		}
		if len(rows) != next.count {
			t.Fatalf("stale session: %d want %d", len(rows), next.count)
		}
	}
	payload["leadership_scope"] = scope("selected", districts[0])
	r = supportJSONRequest(t, h, "PATCH", accountPath, adminToken, payload)
	if r.Code != 200 {
		t.Fatal(r.Body)
	}
	for i, path := range playbackURLs {
		r := supportJSONRequest(t, h, "GET", path, "", nil)
		want := 200
		if i == 1 {
			want = 403
		}
		if r.Code != want {
			t.Fatalf("revoked playback: %d want %d: %s", r.Code, want, r.Body)
		}
	}
	// Future companies are included by all mode, never appended to individual lists.
	var future int
	if err := a.db.QueryRowContext(t.Context(), `INSERT INTO districts(name,coeff) VALUES('Future',1) RETURNING id`).Scan(&future); err != nil {
		t.Fatal(err)
	}
	insertSupportUser(t, a, "future-officer", "FutureOfficer", "", roleOfficer, &future)
	if strings.Contains(request("/api/districts", 200), "Future") {
		t.Fatal("selected scope expanded")
	}
	payload["leadership_scope"] = scope("all")
	r = supportJSONRequest(t, h, "PATCH", accountPath, adminToken, payload)
	if r.Code != 200 {
		t.Fatal(r.Body)
	}
	if !strings.Contains(request("/api/districts", 200), "Future") || !strings.Contains(request("/api/officers", 200), "FutureOfficer") {
		t.Fatal("all scope excludes future company")
	}
	// Missing persisted settings fail closed even when affiliation is populated.
	if _, err := a.db.ExecContext(t.Context(), `UPDATE users SET leadership_scope='' WHERE id=?`, account.ID); err != nil {
		t.Fatal(err)
	}
	if body := request("/api/officers", 200); body != "[]\n" {
		t.Fatalf("missing scope leaked: %s", body)
	}
	request("/api/dashboard?officer_id="+intString(officers[1].ID), 403)
}
