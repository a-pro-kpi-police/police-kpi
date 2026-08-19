package dashboard

import "testing"

func TestBuildReportAggregatesDashboardMetrics(t *testing.T) {
	report := NewPolicy().BuildReport([]Entry{
		{
			Month:            "2026-07",
			OfficerID:        1,
			Quantity:         2,
			CountedQuantity:  1,
			PointsPerUnit:    5,
			RawScore:         10,
			FinalScore:       12.345,
			Status:           StatusConfirmed,
			OfficerName:      "Officer One",
			OfficerRank:      "Rank",
			OfficerDistrict:  "Central",
			CatalogCode:      "A1",
			CatalogIndicator: "Patrol",
			CatalogBlock:     "Block A",
			IsCritical:       true,
		},
		{
			Month:           "2026-07",
			OfficerID:       1,
			Quantity:        1,
			PointsPerUnit:   3,
			FinalScore:      3,
			Status:          StatusPending,
			OfficerDistrict: "Central",
		},
		{
			Month:         "2026-07",
			OfficerID:     2,
			Quantity:      1,
			PointsPerUnit: 4,
			Status:        StatusRejected,
		},
	}, []Officer{
		{ID: 1, FullName: "Officer One", Rank: "Rank", DistrictName: "Central"},
		{ID: 2, FullName: "Officer Two"},
		{ID: 3, FullName: "Officer Three"},
	}, []string{"Block A"}, Filter{})

	if report.TotalFinal != 12.35 {
		t.Fatalf("total final = %v, want 12.35", report.TotalFinal)
	}
	if report.ConfirmedCount != 1 || report.PendingCount != 1 || report.RejectedCount != 1 {
		t.Fatalf("counts = confirmed %d pending %d rejected %d", report.ConfirmedCount, report.PendingCount, report.RejectedCount)
	}
	if report.ConfirmRate != 50 {
		t.Fatalf("confirm rate = %v, want 50", report.ConfirmRate)
	}
	if len(report.Officers) != 3 || report.Officers[0].OfficerID != 1 || report.Officers[0].Final != 12.35 {
		t.Fatalf("officers = %#v", report.Officers)
	}
	if len(report.Blocks) != 1 || report.Blocks[0].Final != 12.35 {
		t.Fatalf("blocks = %#v", report.Blocks)
	}
	volumeByCode := map[string]float64{}
	for _, row := range report.IndicatorVolume {
		volumeByCode[row.Code] = row.Quantity
	}
	if volumeByCode["A1"] != 2 {
		t.Fatalf("indicator volume = %#v", report.IndicatorVolume)
	}
	if report.InactiveOfficers.Count != 1 || report.InactiveOfficers.Total != 3 {
		t.Fatalf("inactive officers = %#v", report.InactiveOfficers)
	}
	if report.MonthlyLimitImpact.Reported != 2 || report.MonthlyLimitImpact.Counted != 1 || report.MonthlyLimitImpact.Excluded != 1 {
		t.Fatalf("monthly limit impact = %#v", report.MonthlyLimitImpact)
	}
	if len(report.ExceededLimits) != 1 || report.ExceededLimits[0].Code != "A1" || report.ExceededLimits[0].AffectedOfficers != 1 {
		t.Fatalf("exceeded limits = %#v", report.ExceededLimits)
	}
	if report.CriticalIndicatorImpact.Records != 1 || report.CriticalIndicatorImpact.Officers != 1 || report.CriticalIndicatorImpact.Quantity != 2 {
		t.Fatalf("critical impact = %#v", report.CriticalIndicatorImpact)
	}
}

func TestBuildComparisonMatchesDashboardClientRules(t *testing.T) {
	officerID := 1
	comparison := NewPolicy().BuildComparison([]Entry{
		{Month: "2026-06", OfficerID: 1, Quantity: 1, FinalScore: 40, Status: StatusConfirmed},
		{Month: "2026-07", OfficerID: 1, Quantity: 1, FinalScore: 60, Status: StatusConfirmed},
		{Month: "2026-07", OfficerID: 2, Quantity: 1, FinalScore: 30, Status: StatusConfirmed},
		{Month: "2026-07", OfficerID: 3, Quantity: 0, FinalScore: 999, Status: StatusConfirmed},
	}, "2026-07", &officerID, []Officer{
		{ID: 1, FullName: "Officer One"},
		{ID: 2, FullName: "Officer Two"},
		{ID: 3, FullName: "Inactive"},
	})

	if comparison.FocusScore != 60 || comparison.CurrentScore != 60 {
		t.Fatalf("scores = (%v, %v), want (60, 60)", comparison.FocusScore, comparison.CurrentScore)
	}
	if comparison.Rank == nil || *comparison.Rank != 1 {
		t.Fatalf("rank = %#v, want 1", comparison.Rank)
	}
	if comparison.OfficerCount != 3 {
		t.Fatalf("officer count = %v, want 3", comparison.OfficerCount)
	}
	if comparison.PeerAverage != 15 {
		t.Fatalf("peer average = %v, want 15", comparison.PeerAverage)
	}
	if comparison.RelativePercent == nil || *comparison.RelativePercent != 300 {
		t.Fatalf("relative percent = %#v, want 300", comparison.RelativePercent)
	}
	if comparison.Baseline != 40 || comparison.PreviousMonthCount != 1 {
		t.Fatalf("baseline = (%v, %v), want (40, 1)", comparison.Baseline, comparison.PreviousMonthCount)
	}
	if len(comparison.History) != 2 || comparison.History[0].Month != "2026-06" || comparison.History[1].Score != 60 {
		t.Fatalf("history = %#v", comparison.History)
	}
}
