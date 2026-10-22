package scoring

import "testing"

func TestMonthlyLimitsPreserveReportedQuantity(t *testing.T) {
	calculation := NewPolicy().Calculate([]Entry{
		confirmedEntry(1, 1, 35, 1, 20, true, false),
		confirmedEntry(2, 2, 8, 5, 6, false, false),
	})

	assertEntryScores(t, calculation.Entries[0], 35, 20, 20)
	assertEntryScores(t, calculation.Entries[1], 8, 6, 30)
	if calculation.Result.ReportedQuantity != 43 {
		t.Fatalf("reported quantity = %v, want 43", calculation.Result.ReportedQuantity)
	}
	if calculation.Result.CountedQuantity != 26 {
		t.Fatalf("counted quantity = %v, want 26", calculation.Result.CountedQuantity)
	}
	if calculation.Result.FinalScore != 50 {
		t.Fatalf("final score = %v, want 50", calculation.Result.FinalScore)
	}
}

func TestMassFilterUsesCappedPositiveScores(t *testing.T) {
	calculation := NewPolicy().Calculate([]Entry{
		confirmedEntry(1, 1, 30, 1, 30, true, false),
		confirmedEntry(2, 2, 6, 5, 6, false, false),
	})

	if calculation.Result.MassShare != 0.5 {
		t.Fatalf("mass share = %v, want 0.5", calculation.Result.MassShare)
	}
	if calculation.Result.AntiCoeff != 0.85 {
		t.Fatalf("anti coeff = %v, want 0.85", calculation.Result.AntiCoeff)
	}
	if calculation.Result.FinalScore != 51 {
		t.Fatalf("final score = %v, want 51", calculation.Result.FinalScore)
	}
}

func TestCriticalIndicatorZeroesMonth(t *testing.T) {
	calculation := NewPolicy().Calculate([]Entry{
		confirmedEntry(1, 1, 5, 10, 10, false, false),
		confirmedEntry(2, 2, 1, -100, 1, false, true),
	})

	if !calculation.Result.Critical {
		t.Fatal("critical = false, want true")
	}
	if calculation.Result.FinalScore != 0 {
		t.Fatalf("final score = %v, want 0", calculation.Result.FinalScore)
	}
	if calculation.Entries[0].FinalScore != 0 || calculation.Entries[1].FinalScore != 0 {
		t.Fatalf("entry final scores = (%v, %v), want (0, 0)", calculation.Entries[0].FinalScore, calculation.Entries[1].FinalScore)
	}
}

func confirmedEntry(id, catalogID int, quantity, points, limit float64, isMass, isCritical bool) Entry {
	return Entry{
		ID:            id,
		CatalogID:     catalogID,
		Quantity:      quantity,
		PointsPerUnit: points,
		Coeff:         1,
		Status:        StatusConfirmed,
		Catalog: &CatalogSnapshot{
			MonthlyLimit: limit,
			IsMass:       isMass,
			IsCritical:   isCritical,
		},
	}
}

func assertEntryScores(t *testing.T, entry Entry, quantity, counted, capped float64) {
	t.Helper()
	if entry.Quantity != quantity || entry.CountedQuantity != counted || entry.CappedScore != capped {
		t.Fatalf("entry %d scores = (%v, %v, %v), want (%v, %v, %v)", entry.ID, entry.Quantity, entry.CountedQuantity, entry.CappedScore, quantity, counted, capped)
	}
}
