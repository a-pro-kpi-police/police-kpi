package main

import (
	"testing"

	domainusers "kpi-police/backend/internal/domain/users"
)

func TestRatingOfficersOutSortsAndIncludesEfficiency(t *testing.T) {
	districtID := 2
	rows := ratingOfficersOut([]domainusers.OfficerWithEfficiency{
		{
			User: domainusers.User{
				ID:         1,
				FullName:   "Officer One",
				Rank:       "Rank",
				DistrictID: &districtID,
				District:   &domainusers.District{ID: districtID, Name: "Central"},
			},
			Efficiency: domainusers.Efficiency{Rank: 2, Total: 2, Score: 80},
		},
		{
			User:       domainusers.User{ID: 2, FullName: "Officer Two", Rank: "Rank"},
			Efficiency: domainusers.Efficiency{Rank: 1, Total: 2, Score: 100},
		},
	}, map[int]ratingScore{
		1: {Final: 25, Confirmed: 3, Pending: 1},
		2: {Final: 40, Confirmed: 4},
	})

	if len(rows) != 2 {
		t.Fatalf("rows length = %d, want 2", len(rows))
	}
	if rows[0].OfficerID != 2 || rows[0].Final != 40 {
		t.Fatalf("first row = %#v, want officer 2 with score 40", rows[0])
	}
	if rows[1].DistrictID != districtID || rows[1].District != "Central" {
		t.Fatalf("district = (%d, %q), want (2, Central)", rows[1].DistrictID, rows[1].District)
	}
	if rows[1].EfficiencyRank != 2 || rows[1].EfficiencyTotal != 2 || rows[1].EfficiencyScore != 80 {
		t.Fatalf("efficiency = (%d, %d, %v), want (2, 2, 80)",
			rows[1].EfficiencyRank,
			rows[1].EfficiencyTotal,
			rows[1].EfficiencyScore,
		)
	}
}
