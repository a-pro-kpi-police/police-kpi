package main

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
)

type adminCatalogFields struct {
	Code          string
	Indicator     string
	Block         string
	Subblock      string
	NPA           string
	PointsPerUnit float64
	Unit          string
	SourceURL     string
	MonthlyLimit  float64
	IsMass        bool
	IsCritical    bool
}

func (a *App) handleAdminListCatalog(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAdmin(a, w, r); !ok {
		return
	}
	items, err := a.catalogRepository().ListActivePositive(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, catalogItems(items))
}

func (a *App) handleAdminCreateCatalogItem(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAdmin(a, w, r); !ok {
		return
	}
	var payload AdminCatalogSavePayload
	if err := readJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Некорректные данные KPI")
		return
	}
	fields, err := adminCatalogFieldsFromPayload(payload)
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	id, err := a.saveAdminCatalogItem(r.Context(), 0, fields)
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	item, err := a.catalogRepository().Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, catalogItem(item))
}

func (a *App) handleAdminUpdateCatalogItem(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAdmin(a, w, r); !ok {
		return
	}
	catalogID, ok := parseIDPathValue(r, "catalog_id")
	if !ok {
		writeError(w, http.StatusNotFound, "Показатель KPI не найден")
		return
	}
	var payload AdminCatalogSavePayload
	if err := readJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Некорректные данные KPI")
		return
	}
	fields, err := adminCatalogFieldsFromPayload(payload)
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	id, err := a.saveAdminCatalogItem(r.Context(), catalogID, fields)
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	item, err := a.catalogRepository().Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, catalogItem(item))
}

func (a *App) handleAdminDeleteCatalogItem(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAdmin(a, w, r); !ok {
		return
	}
	catalogID, ok := parseIDPathValue(r, "catalog_id")
	if !ok {
		writeError(w, http.StatusNotFound, "Показатель KPI не найден")
		return
	}
	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(r.Context(), `UPDATE kpi_catalog SET is_active = 0 WHERE id = ? AND is_active = 1`, catalogID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if affected == 0 {
		writeError(w, http.StatusNotFound, "Показатель KPI не найден")
		return
	}
	if _, err := tx.ExecContext(r.Context(), `DELETE FROM kpi_targets WHERE catalog_id = ?`, catalogID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) saveAdminCatalogItem(ctx context.Context, catalogID int, fields adminCatalogFields) (int, error) {
	if catalogID > 0 {
		if err := a.ensureAdminCatalogCodeUnique(ctx, fields.Code, catalogID); err != nil {
			return 0, err
		}
		return a.updateAdminCatalogItem(ctx, catalogID, fields)
	}

	existingID, active, err := a.catalogCodeStatus(ctx, fields.Code)
	if err != nil {
		return 0, err
	}
	if existingID > 0 {
		if active {
			return 0, httpErr{status: http.StatusConflict, detail: "Код KPI уже занят"}
		}
		return a.updateAdminCatalogItem(ctx, existingID, fields)
	}

	var id int
	err = a.db.QueryRowContext(ctx, `INSERT INTO kpi_catalog
			(code, indicator, block, subblock, npa, points_per_unit, unit, source_url, monthly_limit, is_mass, is_critical, is_active)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)
			RETURNING id`,
		fields.Code,
		fields.Indicator,
		fields.Block,
		fields.Subblock,
		fields.NPA,
		fields.PointsPerUnit,
		fields.Unit,
		fields.SourceURL,
		fields.MonthlyLimit,
		boolInt(fields.IsMass),
		boolInt(fields.IsCritical),
	).Scan(&id)
	return id, err
}

func (a *App) updateAdminCatalogItem(ctx context.Context, catalogID int, fields adminCatalogFields) (int, error) {
	result, err := a.db.ExecContext(ctx, `UPDATE kpi_catalog SET
			code = ?, indicator = ?, block = ?, subblock = ?, npa = ?,
			points_per_unit = ?, unit = ?, source_url = ?, monthly_limit = ?,
			is_mass = ?, is_critical = ?, is_active = 1
		WHERE id = ?`,
		fields.Code,
		fields.Indicator,
		fields.Block,
		fields.Subblock,
		fields.NPA,
		fields.PointsPerUnit,
		fields.Unit,
		fields.SourceURL,
		fields.MonthlyLimit,
		boolInt(fields.IsMass),
		boolInt(fields.IsCritical),
		catalogID,
	)
	if err != nil {
		return 0, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if affected == 0 {
		return 0, httpErr{status: http.StatusNotFound, detail: "Показатель KPI не найден"}
	}
	return catalogID, nil
}

func (a *App) catalogCodeStatus(ctx context.Context, code string) (id int, active bool, err error) {
	var isActive int
	err = a.db.QueryRowContext(ctx, `SELECT id, is_active FROM kpi_catalog WHERE code = ?`, code).Scan(&id, &isActive)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return id, isActive != 0, nil
}

func adminCatalogFieldsFromPayload(payload AdminCatalogSavePayload) (adminCatalogFields, error) {
	code := strings.ToUpper(strings.TrimSpace(payload.Code))
	if code == "" {
		return adminCatalogFields{}, httpErr{status: http.StatusBadRequest, detail: "Код KPI обязателен"}
	}
	indicator := strings.TrimSpace(payload.Indicator)
	if indicator == "" {
		return adminCatalogFields{}, httpErr{status: http.StatusBadRequest, detail: "Показатель KPI обязателен"}
	}
	block := strings.TrimSpace(payload.Block)
	if block == "" {
		return adminCatalogFields{}, httpErr{status: http.StatusBadRequest, detail: "Блок KPI обязателен"}
	}
	if payload.PointsPerUnit == nil || *payload.PointsPerUnit < 0 {
		return adminCatalogFields{}, httpErr{status: http.StatusBadRequest, detail: "Балл KPI должен быть нулём или больше"}
	}
	monthlyLimit := 0.0
	if payload.MonthlyLimit != nil {
		monthlyLimit = *payload.MonthlyLimit
	}
	if monthlyLimit < 0 {
		return adminCatalogFields{}, httpErr{status: http.StatusBadRequest, detail: "Лимит KPI не может быть отрицательным"}
	}
	unit := strings.TrimSpace(payload.Unit)
	if unit == "" {
		unit = "ед."
	}
	return adminCatalogFields{
		Code:          code,
		Indicator:     indicator,
		Block:         block,
		Subblock:      strings.TrimSpace(payload.Subblock),
		NPA:           strings.TrimSpace(payload.NPA),
		PointsPerUnit: *payload.PointsPerUnit,
		Unit:          unit,
		SourceURL:     strings.TrimSpace(payload.SourceURL),
		MonthlyLimit:  monthlyLimit,
		IsMass:        payload.IsMass,
		IsCritical:    payload.IsCritical,
	}, nil
}

func (a *App) ensureAdminCatalogCodeUnique(ctx context.Context, code string, excludeID int) error {
	taken, err := valueTaken(ctx, a.db, `SELECT id FROM kpi_catalog WHERE code = ? AND id <> ? LIMIT 1`, code, excludeID)
	if err != nil {
		return err
	}
	if taken {
		return httpErr{status: http.StatusConflict, detail: "Код KPI уже занят"}
	}
	return nil
}
