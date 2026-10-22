package main

import (
	"net/http"

	domaindashboard "kpi-police/backend/internal/domain/dashboard"
)

func (a *App) handleDashboard(w http.ResponseWriter, r *http.Request) {
	user, ok := requireHeadViewer(a, w, r)
	if !ok {
		return
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
	if user.Role == roleLeadership && officerID != nil {
		if err := a.ensureOfficerInHeadScope(r.Context(), user, *officerID); err != nil {
			writeHTTPError(w, err)
			return
		}
	}
	a.writeDashboard(w, r, domaindashboard.Filter{
		Month:      r.URL.Query().Get("month"),
		OfficerID:  officerID,
		DistrictID: districtID,
	}, leadershipAverageScope(user))
}

func (a *App) handleAdminOfficerDashboard(w http.ResponseWriter, r *http.Request) {
	officerID, ok := a.requireAdminOfficerPathID(w, r)
	if !ok {
		return
	}
	a.writeDashboard(w, r, domaindashboard.Filter{
		Month:     r.URL.Query().Get("month"),
		OfficerID: &officerID,
	}, 0)
}

func (a *App) writeDashboard(w http.ResponseWriter, r *http.Request, filter domaindashboard.Filter, averageScope int) {
	report, err := a.dashboardService().BuildDashboard(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	average, err := a.globalKPIAverage(r.Context(), filter.Month, averageScope)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	response := dashboardOut(report)
	response["global_average_score"] = average
	writeJSON(w, http.StatusOK, response)
}

func dashboardOut(report domaindashboard.Report) map[string]any {
	return map[string]any{
		"total_final":                  report.TotalFinal,
		"confirmed_count":              report.ConfirmedCount,
		"pending_count":                report.PendingCount,
		"rejected_count":               report.RejectedCount,
		"officers_count":               report.OfficersCount,
		"confirm_rate":                 report.ConfirmRate,
		"confirm_rate_confirmed_count": report.ConfirmRateConfirmedCount,
		"confirm_rate_decided_count":   report.ConfirmRateDecidedCount,
		"officers":                     dashboardOfficersOut(report.Officers),
		"blocks":                       dashboardBlocksOut(report.Blocks),
		"trend":                        dashboardTrendOut(report.Trend),
		"status_trend":                 dashboardStatusTrendOut(report.StatusTrend),
		"indicator_volume":             dashboardIndicatorVolumeOut(report.IndicatorVolume),
		"district_review":              dashboardDistrictReviewOut(report.DistrictReview),
		"comparisons":                  dashboardComparisonOut(report.Comparisons),
		"inactive_officers":            dashboardInactiveOfficersOut(report.InactiveOfficers),
		"monthly_limit_impact":         dashboardMonthlyLimitImpactOut(report.MonthlyLimitImpact),
		"exceeded_limits":              dashboardExceededLimitsOut(report.ExceededLimits),
		"critical_indicator_impact":    dashboardCriticalIndicatorImpactOut(report.CriticalIndicatorImpact),
	}
}

func dashboardComparisonOut(comparison domaindashboard.Comparison) map[string]any {
	return map[string]any{
		"focusScore":         comparison.FocusScore,
		"currentScore":       comparison.CurrentScore,
		"baseline":           comparison.Baseline,
		"previousMonthCount": comparison.PreviousMonthCount,
		"currentMonth":       comparison.CurrentMonth,
		"rank":               comparison.Rank,
		"officerCount":       comparison.OfficerCount,
		"peerAverage":        comparison.PeerAverage,
		"relativePercent":    comparison.RelativePercent,
		"history":            dashboardComparisonHistoryOut(comparison.History),
	}
}

func dashboardComparisonHistoryOut(rows []domaindashboard.ComparisonHistoryRow) []map[string]any {
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		result = append(result, map[string]any{"month": row.Month, "score": row.Score})
	}
	return result
}

func dashboardInactiveOfficersOut(inactive domaindashboard.InactiveOfficers) map[string]any {
	return map[string]any{"count": inactive.Count, "total": inactive.Total}
}

func dashboardMonthlyLimitImpactOut(impact domaindashboard.MonthlyLimitImpact) map[string]any {
	return map[string]any{
		"reported":          impact.Reported,
		"counted":           impact.Counted,
		"excluded":          impact.Excluded,
		"affected_records":  impact.AffectedRecords,
		"affected_officers": impact.AffectedOfficers,
	}
}

func dashboardExceededLimitsOut(rows []domaindashboard.ExceededLimitRow) []map[string]any {
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		result = append(result, map[string]any{
			"code":              row.Code,
			"indicator":         row.Indicator,
			"reported":          row.Reported,
			"counted":           row.Counted,
			"excluded":          row.Excluded,
			"affected_officers": row.AffectedOfficers,
		})
	}
	return result
}

func dashboardCriticalIndicatorImpactOut(impact domaindashboard.CriticalIndicatorImpact) map[string]any {
	return map[string]any{
		"records":  impact.Records,
		"officers": impact.Officers,
		"quantity": impact.Quantity,
	}
}

func dashboardOfficersOut(officers []domaindashboard.OfficerRow) []map[string]any {
	result := make([]map[string]any, 0, len(officers))
	for _, officer := range officers {
		result = append(result, map[string]any{
			"officer_id": officer.OfficerID,
			"name":       officer.Name,
			"rank":       officer.Rank,
			"district":   officer.District,
			"final":      officer.Final,
			"raw":        officer.Raw,
			"confirmed":  officer.Confirmed,
			"pending":    officer.Pending,
			"blocks":     officer.Blocks,
		})
	}
	return result
}

func dashboardBlocksOut(blocks []domaindashboard.BlockRow) []map[string]any {
	result := make([]map[string]any, 0, len(blocks))
	for _, block := range blocks {
		result = append(result, map[string]any{"block": block.Block, "final": block.Final})
	}
	return result
}

func dashboardTrendOut(trend []domaindashboard.TrendRow) []map[string]any {
	result := make([]map[string]any, 0, len(trend))
	for _, row := range trend {
		result = append(result, map[string]any{"month": row.Month, "final": row.Final})
	}
	return result
}

func dashboardStatusTrendOut(trend []domaindashboard.StatusTrendRow) []map[string]any {
	result := make([]map[string]any, 0, len(trend))
	for _, row := range trend {
		result = append(result, map[string]any{
			"month":     row.Month,
			"confirmed": row.Confirmed,
			"pending":   row.Pending,
			"rejected":  row.Rejected,
		})
	}
	return result
}

func dashboardIndicatorVolumeOut(rows []domaindashboard.IndicatorVolumeRow) []map[string]any {
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		result = append(result, map[string]any{
			"code":      row.Code,
			"indicator": row.Indicator,
			"quantity":  row.Quantity,
		})
	}
	return result
}

func dashboardDistrictReviewOut(rows []domaindashboard.DistrictReviewRow) []map[string]any {
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		result = append(result, map[string]any{
			"district": row.District,
			"pending":  row.Pending,
			"rejected": row.Rejected,
		})
	}
	return result
}
