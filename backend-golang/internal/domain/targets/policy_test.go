package targets

import "testing"

func TestTargetQuantitiesValidatesCatalogAndQuantity(t *testing.T) {
	quantities, err := NewPolicy().TargetQuantities([]SaveItem{
		{CatalogID: 1, TargetQuantity: 10},
		{CatalogID: 2, TargetQuantity: 0},
	}, map[int]bool{1: true, 2: true})
	if err != nil {
		t.Fatal(err)
	}
	if quantities[1] != 10 || quantities[2] != 0 {
		t.Fatalf("quantities = %#v, want catalog 1 = 10 and catalog 2 = 0", quantities)
	}
}

func TestTargetQuantitiesRejectsNegativeQuantity(t *testing.T) {
	_, err := NewPolicy().TargetQuantities([]SaveItem{
		{CatalogID: 1, TargetQuantity: -1},
	}, map[int]bool{1: true})
	if err != ErrNegativeQuantity {
		t.Fatalf("error = %v, want %v", err, ErrNegativeQuantity)
	}
}

func TestValidateMonthRejectsInvalidMonth(t *testing.T) {
	if err := NewPolicy().ValidateMonth("2026-13"); err != ErrInvalidMonth {
		t.Fatalf("error = %v, want %v", err, ErrInvalidMonth)
	}
}

func TestBuildProgressCalculatesItemsAndTotals(t *testing.T) {
	policy := NewPolicy()

	progress := policy.BuildProgress("2026-07", 7, 100, []Target{
		{CatalogID: 1, Block: "A", TargetQuantity: 10, TargetScore: 50},
		{CatalogID: 2, Block: "B", TargetQuantity: 5, TargetScore: 25},
	}, map[int]Actual{
		1: {Quantity: 7, CountedQuantity: 6, Score: 30},
		2: {Quantity: 8, CountedQuantity: 5, Score: 25},
	})

	if progress.ActualScore != 55 {
		t.Fatalf("actual score = %v, want 55", progress.ActualScore)
	}
	if progress.TargetScore != 100 {
		t.Fatalf("target score = %v, want 100", progress.TargetScore)
	}
	if progress.AchievementPercent != 55 {
		t.Fatalf("achievement = %v, want 55", progress.AchievementPercent)
	}
	if progress.Items[0].RemainingQuantity != 3 {
		t.Fatalf("remaining quantity = %v, want 3", progress.Items[0].RemainingQuantity)
	}
	if progress.Items[1].RemainingQuantity != 0 {
		t.Fatalf("remaining quantity = %v, want 0", progress.Items[1].RemainingQuantity)
	}
}

func TestBuildProgressDefaultsEmptyNormToOneHundred(t *testing.T) {
	progress := NewPolicy().BuildProgress("2026-07", 7, 0, nil, nil)
	if progress.TargetScore != 100 {
		t.Fatalf("target score = %v, want 100", progress.TargetScore)
	}
}

func TestBuildOverviewSortsAndAggregates(t *testing.T) {
	policy := NewPolicy()
	overview := policy.BuildOverview("2026-07", []OverviewInput{
		{
			OfficerID: 1,
			Name:      "On Target",
			Progress:  Progress{ActualScore: 100, TargetScore: 100, AchievementPercent: 100, Items: []ProgressItem{{Target: Target{Block: "A", TargetScore: 50}, ConfirmedScore: 50}}},
			Penalty:   Penalty{Score: -5, Count: 1},
		},
		{
			OfficerID: 2,
			Name:      "Behind",
			Progress:  Progress{ActualScore: 40, TargetScore: 100, AchievementPercent: 40, Items: []ProgressItem{{Target: Target{Block: "A", TargetScore: 50}, ConfirmedScore: 20}}},
		},
	}, false)

	if overview.OnTargetCount != 1 {
		t.Fatalf("on target count = %v, want 1", overview.OnTargetCount)
	}
	if overview.Officers[0].Status != StatusBehind {
		t.Fatalf("first status = %q, want behind", overview.Officers[0].Status)
	}
	if overview.Officers[1].Status != StatusOnTarget {
		t.Fatalf("second status = %q, want on_target", overview.Officers[1].Status)
	}
	if overview.PenaltyScore != -5 || overview.PenaltyCount != 1 {
		t.Fatalf("penalty = (%v, %v), want (-5, 1)", overview.PenaltyScore, overview.PenaltyCount)
	}
	if len(overview.Blocks) != 1 || overview.Blocks[0].ActualScore != 70 || overview.Blocks[0].TargetScore != 100 {
		t.Fatalf("blocks = %+v, want one A block with actual 70 target 100", overview.Blocks)
	}
}
