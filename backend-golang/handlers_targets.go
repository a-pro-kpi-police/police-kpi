package main

import (
	"context"
	"errors"
	"net/http"

	apptargets "kpi-police/backend-golang/internal/application/targets"
	domaintargets "kpi-police/backend-golang/internal/domain/targets"
)

func (a *App) handleGetTargets(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(a, w, r)
	if !ok {
		return
	}
	officerID, ok := a.resolveTargetOfficerID(w, r, user, true)
	if !ok {
		return
	}
	rows, err := a.targetQueryService().TargetRows(r.Context(), officerID, r.URL.Query().Get("month"))
	if err != nil {
		writeHTTPError(w, targetQueryHTTPError(err))
		return
	}
	writeJSON(w, http.StatusOK, targetRowsOut(rows))
}

func (a *App) handleSaveTargets(w http.ResponseWriter, r *http.Request) {
	user, ok := requireHead(a, w, r)
	if !ok {
		return
	}
	var payload TargetSavePayload
	if err := readJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Некорректные данные целей")
		return
	}
	if err := a.ensureOfficerInHeadScope(r.Context(), user, payload.OfficerID); err != nil {
		writeHTTPError(w, err)
		return
	}
	if err := a.saveOfficerTargets(r.Context(), payload); err != nil {
		writeHTTPError(w, err)
		return
	}
	rows, err := a.targetQueryService().TargetRows(r.Context(), payload.OfficerID, payload.Month)
	if err != nil {
		writeHTTPError(w, targetQueryHTTPError(err))
		return
	}
	writeJSON(w, http.StatusOK, targetRowsOut(rows))
}

func (a *App) handleGetGroupTargets(w http.ResponseWriter, r *http.Request) {
	user, ok := requireHead(a, w, r)
	if !ok {
		return
	}
	districtID, err := parseInt(r.URL.Query().Get("district_id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "Группа не найдена")
		return
	}
	scopedDistrictID, err := scopedHeadDistrictID(user, &districtID)
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	if scopedDistrictID != nil {
		districtID = *scopedDistrictID
	}
	rows, err := a.targetQueryService().GroupTargetRows(r.Context(), districtID, r.URL.Query().Get("month"))
	if err != nil {
		writeHTTPError(w, targetQueryHTTPError(err))
		return
	}
	writeJSON(w, http.StatusOK, groupTargetRowsOut(rows))
}

func (a *App) handleSaveGroupTargets(w http.ResponseWriter, r *http.Request) {
	user, ok := requireHead(a, w, r)
	if !ok {
		return
	}
	var payload GroupTargetSavePayload
	if err := readJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Некорректные данные групповых целей")
		return
	}
	scopedDistrictID, err := scopedHeadDistrictID(user, &payload.DistrictID)
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	if scopedDistrictID != nil {
		payload.DistrictID = *scopedDistrictID
	}
	if err := a.saveGroupTargets(r.Context(), payload); err != nil {
		writeHTTPError(w, err)
		return
	}
	rows, err := a.targetQueryService().GroupTargetRows(r.Context(), payload.DistrictID, payload.Month)
	if err != nil {
		writeHTTPError(w, targetQueryHTTPError(err))
		return
	}
	writeJSON(w, http.StatusOK, groupTargetRowsOut(rows))
}

func (a *App) handleTargetProgress(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(a, w, r)
	if !ok {
		return
	}
	selectedMonth := r.URL.Query().Get("month")
	if selectedMonth == "" {
		selectedMonth = currentMonth()
	}
	officerID, ok := a.resolveTargetOfficerID(w, r, user, true)
	if !ok {
		return
	}
	progress, err := a.targetQueryService().Progress(r.Context(), officerID, selectedMonth)
	if err != nil {
		writeHTTPError(w, targetQueryHTTPError(err))
		return
	}
	writeJSON(w, http.StatusOK, targetProgressOut(progress))
}

func (a *App) handleTargetOverview(w http.ResponseWriter, r *http.Request) {
	user, ok := requireHead(a, w, r)
	if !ok {
		return
	}
	selectedMonth := r.URL.Query().Get("month")
	if selectedMonth == "" {
		selectedMonth = currentMonth()
	}
	officerID, err := parseOptionalInt(r.URL.Query().Get("officer_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Сотрудник указан неверно")
		return
	}
	districtID, err := parseOptionalInt(r.URL.Query().Get("district_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Группа указана неверно")
		return
	}
	districtID, err = scopedHeadDistrictID(user, districtID)
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	if officerID != nil {
		if err := a.ensureOfficerInHeadScope(r.Context(), user, *officerID); err != nil {
			writeHTTPError(w, err)
			return
		}
	}
	a.writeTargetOverview(w, r, domaintargets.OverviewFilter{
		Month:      selectedMonth,
		OfficerID:  officerID,
		DistrictID: districtID,
	})
}

func (a *App) handleAdminOfficerTargetOverview(w http.ResponseWriter, r *http.Request) {
	officerID, ok := a.requireAdminOfficerPathID(w, r)
	if !ok {
		return
	}
	selectedMonth := r.URL.Query().Get("month")
	if selectedMonth == "" {
		selectedMonth = currentMonth()
	}
	a.writeTargetOverview(w, r, domaintargets.OverviewFilter{
		Month:     selectedMonth,
		OfficerID: &officerID,
	})
}

func (a *App) writeTargetOverview(w http.ResponseWriter, r *http.Request, filter domaintargets.OverviewFilter) {
	overview, err := a.targetQueryService().Overview(r.Context(), filter)
	if err != nil {
		writeHTTPError(w, targetQueryHTTPError(err))
		return
	}
	writeJSON(w, http.StatusOK, targetOverviewOut(overview))
}

func (a *App) resolveTargetOfficerID(w http.ResponseWriter, r *http.Request, user *User, requireHeadParam bool) (int, bool) {
	if isEntryManagerRole(user.Role) {
		officerID, err := parseInt(r.URL.Query().Get("officer_id"))
		if err != nil || officerID <= 0 {
			writeError(w, http.StatusBadRequest, "Сотрудник не указан")
			return 0, false
		}
		if err := a.ensureOfficerInHeadScope(r.Context(), user, officerID); err != nil {
			writeHTTPError(w, err)
			return 0, false
		}
		return officerID, true
	}
	if value := r.URL.Query().Get("officer_id"); value != "" && requireHeadParam {
		officerID, err := parseInt(value)
		if err != nil || officerID != user.ID {
			writeError(w, http.StatusForbidden, "Недостаточно прав")
			return 0, false
		}
	}
	return user.ID, true
}

func (a *App) saveOfficerTargets(ctx context.Context, payload TargetSavePayload) error {
	service := apptargets.NewService(a.targetRepository(), domaintargets.NewPolicy())
	if err := service.SaveOfficerTargets(ctx, payload.OfficerID, payload.Month, domainTargetSaveItems(payload.Items)); err != nil {
		return targetSaveHTTPError(err)
	}
	return nil
}

func (a *App) saveGroupTargets(ctx context.Context, payload GroupTargetSavePayload) error {
	service := apptargets.NewService(a.targetRepository(), domaintargets.NewPolicy())
	if err := service.SaveGroupTargets(ctx, payload.DistrictID, payload.Month, domainTargetSaveItems(payload.Items)); err != nil {
		return targetSaveHTTPError(err)
	}
	return nil
}

func domainTargetSaveItems(items []TargetSaveItem) []domaintargets.SaveItem {
	result := make([]domaintargets.SaveItem, 0, len(items))
	for _, item := range items {
		result = append(result, domaintargets.SaveItem{
			CatalogID:      item.CatalogID,
			TargetQuantity: item.TargetQuantity,
		})
	}
	return result
}

func targetSaveHTTPError(err error) error {
	switch {
	case errors.Is(err, domaintargets.ErrInvalidMonth):
		return httpErr{status: http.StatusBadRequest, detail: "Месяц указан неверно"}
	case errors.Is(err, domaintargets.ErrOfficerNotFound):
		return httpErr{status: http.StatusNotFound, detail: "Сотрудник не найден"}
	case errors.Is(err, domaintargets.ErrDistrictNotFound):
		return httpErr{status: http.StatusNotFound, detail: "Группа не найдена"}
	case errors.Is(err, domaintargets.ErrEmptyGroup):
		return httpErr{status: http.StatusBadRequest, detail: "В группе нет сотрудников"}
	case errors.Is(err, domaintargets.ErrCatalogNotFound):
		return httpErr{status: http.StatusNotFound, detail: "Показатель KPI не найден"}
	case errors.Is(err, domaintargets.ErrNegativeQuantity):
		return httpErr{status: http.StatusBadRequest, detail: "Цель не может быть отрицательной"}
	default:
		return err
	}
}

func targetQueryHTTPError(err error) error {
	switch {
	case errors.Is(err, domaintargets.ErrInvalidMonth):
		return httpErr{status: http.StatusBadRequest, detail: "Месяц указан неверно"}
	case errors.Is(err, domaintargets.ErrOfficerNotFound):
		return httpErr{status: http.StatusNotFound, detail: "Сотрудник не найден"}
	case errors.Is(err, domaintargets.ErrDistrictNotFound):
		return httpErr{status: http.StatusNotFound, detail: "Группа не найдена"}
	default:
		return err
	}
}

func targetRowsOut(rows []domaintargets.Target) []TargetOut {
	result := make([]TargetOut, 0, len(rows))
	for _, row := range rows {
		result = append(result, targetRowOut(row))
	}
	return result
}

func targetRowOut(row domaintargets.Target) TargetOut {
	return TargetOut{
		CatalogID:      row.CatalogID,
		Code:           row.Code,
		Indicator:      row.Indicator,
		Block:          row.Block,
		Unit:           row.Unit,
		PointsPerUnit:  row.PointsPerUnit,
		MonthlyLimit:   row.MonthlyLimit,
		IsMass:         row.IsMass,
		IsCritical:     row.IsCritical,
		TargetQuantity: row.TargetQuantity,
		TargetScore:    row.TargetScore,
	}
}

func groupTargetRowsOut(rows []domaintargets.GroupTarget) []GroupTargetOut {
	result := make([]GroupTargetOut, 0, len(rows))
	for _, row := range rows {
		result = append(result, GroupTargetOut{
			TargetOut: targetRowOut(row.Target),
			Mixed:     row.Mixed,
		})
	}
	return result
}

func targetProgressOut(progress domaintargets.Progress) TargetProgressOut {
	return TargetProgressOut{
		Month:              progress.Month,
		OfficerID:          progress.OfficerID,
		ActualScore:        progress.ActualScore,
		TargetScore:        progress.TargetScore,
		AchievementPercent: progress.AchievementPercent,
		Items:              targetProgressItemsOut(progress.Items),
	}
}

func targetProgressItemsOut(items []domaintargets.ProgressItem) []TargetProgressItemOut {
	result := make([]TargetProgressItemOut, 0, len(items))
	for _, item := range items {
		result = append(result, TargetProgressItemOut{
			CatalogID:          item.CatalogID,
			Code:               item.Code,
			Indicator:          item.Indicator,
			Block:              item.Block,
			Unit:               item.Unit,
			PointsPerUnit:      item.PointsPerUnit,
			MonthlyLimit:       item.MonthlyLimit,
			IsMass:             item.IsMass,
			IsCritical:         item.IsCritical,
			TargetQuantity:     item.TargetQuantity,
			ConfirmedQuantity:  item.ConfirmedQuantity,
			CountedQuantity:    item.CountedQuantity,
			RemainingQuantity:  item.RemainingQuantity,
			TargetScore:        item.TargetScore,
			ConfirmedScore:     item.ConfirmedScore,
			AchievementPercent: item.AchievementPercent,
		})
	}
	return result
}

func targetOverviewOut(overview domaintargets.Overview) TargetOverviewOut {
	return TargetOverviewOut{
		Month:              overview.Month,
		ActualScore:        overview.ActualScore,
		TargetScore:        overview.TargetScore,
		AchievementPercent: overview.AchievementPercent,
		OnTargetCount:      overview.OnTargetCount,
		OfficersCount:      overview.OfficersCount,
		PenaltyScore:       overview.PenaltyScore,
		PenaltyCount:       overview.PenaltyCount,
		Officers:           targetOverviewOfficersOut(overview.Officers),
		Blocks:             targetBlockProgressOut(overview.Blocks),
		Items:              targetProgressItemsOut(overview.Items),
	}
}

func targetOverviewOfficersOut(officers []domaintargets.OfficerOverview) []TargetOverviewOfficerOut {
	result := make([]TargetOverviewOfficerOut, 0, len(officers))
	for _, officer := range officers {
		result = append(result, TargetOverviewOfficerOut{
			OfficerID:          officer.OfficerID,
			Name:               officer.Name,
			ActualScore:        officer.ActualScore,
			TargetScore:        officer.TargetScore,
			AchievementPercent: officer.AchievementPercent,
			Gap:                officer.Gap,
			Status:             officer.Status,
			PenaltyScore:       officer.PenaltyScore,
			PenaltyCount:       officer.PenaltyCount,
		})
	}
	return result
}

func targetBlockProgressOut(blocks []domaintargets.BlockProgress) []TargetBlockProgressOut {
	result := make([]TargetBlockProgressOut, 0, len(blocks))
	for _, block := range blocks {
		result = append(result, TargetBlockProgressOut{
			Block:              block.Block,
			ActualScore:        block.ActualScore,
			TargetScore:        block.TargetScore,
			AchievementPercent: block.AchievementPercent,
		})
	}
	return result
}
