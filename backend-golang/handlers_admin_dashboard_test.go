package main

import (
	"context"
	"net/http"
	"testing"
)

func TestAdminOfficerDashboardRoutesEnforceRoleAndOfficerID(t *testing.T) {
	app := newSupportTestApp(t)
	defer app.db.Close()
	handler := app.routes()

	districtID := insertSupportDistrict(t, app)
	selected := insertSupportUser(t, app, "selected.dashboard", "Выбранный Сотрудник", "Капитан полиции", roleOfficer, &districtID)
	other := insertSupportUser(t, app, "other.dashboard", "Другой Сотрудник", "Лейтенант полиции", roleOfficer, &districtID)
	head := insertSupportUser(t, app, "head.dashboard", "Начальник", "Майор полиции", roleHead, &districtID)
	admin, err := app.getUserByUsername(context.Background(), "admin")
	if err != nil {
		t.Fatal(err)
	}

	adminToken := supportToken(t, app, admin)
	headToken := supportToken(t, app, head)
	officerToken := supportToken(t, app, selected)
	dashboardPath := "/api/admin/officers/" + intString(selected.ID) + "/dashboard"

	response := supportJSONRequest(t, handler, http.MethodGet, dashboardPath, adminToken, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("admin dashboard status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	var dashboard struct {
		Officers []struct {
			OfficerID int    `json:"officer_id"`
			Name      string `json:"name"`
		} `json:"officers"`
	}
	decodeSupportResponse(t, response, &dashboard)
	if len(dashboard.Officers) != 1 || dashboard.Officers[0].OfficerID != selected.ID || dashboard.Officers[0].Name != selected.FullName {
		t.Fatalf("dashboard officers = %#v, want only selected officer %d", dashboard.Officers, selected.ID)
	}
	for name, token := range map[string]string{"head": headToken, "officer": officerToken} {
		response = supportJSONRequest(t, handler, http.MethodGet, dashboardPath, token, nil)
		if response.Code != http.StatusForbidden {
			t.Fatalf("%s admin dashboard status = %d, want %d", name, response.Code, http.StatusForbidden)
		}
	}

	response = supportJSONRequest(t, handler, http.MethodGet, "/api/dashboard?officer_id="+intString(selected.ID), adminToken, nil)
	if response.Code != http.StatusForbidden {
		t.Fatalf("admin head-dashboard status = %d, want %d", response.Code, http.StatusForbidden)
	}

	for _, path := range []string{
		"/api/admin/officers/not-a-number/dashboard",
		"/api/admin/officers/" + intString(head.ID) + "/dashboard",
		"/api/admin/officers/" + intString(other.ID+10000) + "/dashboard",
	} {
		response = supportJSONRequest(t, handler, http.MethodGet, path, adminToken, nil)
		if response.Code != http.StatusNotFound {
			t.Fatalf("invalid officer path %q status = %d, want %d", path, response.Code, http.StatusNotFound)
		}
	}

	targetPath := "/api/admin/officers/" + intString(selected.ID) + "/target-overview?month=2026-09"
	response = supportJSONRequest(t, handler, http.MethodGet, targetPath, adminToken, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("admin target overview status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	var overview TargetOverviewOut
	decodeSupportResponse(t, response, &overview)
	if overview.OfficersCount != 1 {
		t.Fatalf("target overview officers count = %d, want 1", overview.OfficersCount)
	}
}
