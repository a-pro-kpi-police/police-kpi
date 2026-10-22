package users

import (
	"math"
	"sort"
)

type Efficiency struct {
	Rank  int
	Total int
	Score float64
}

type OfficerWithEfficiency struct {
	User       User
	Efficiency Efficiency
}

type EfficiencyPolicy struct{}

func NewEfficiencyPolicy() EfficiencyPolicy {
	return EfficiencyPolicy{}
}

func (EfficiencyPolicy) RankOfficers(officers []User, scores map[int]float64) ([]OfficerWithEfficiency, map[int]Efficiency) {
	ranked := make([]User, len(officers))
	copy(ranked, officers)
	sort.SliceStable(ranked, func(i, j int) bool {
		leftScore := scores[ranked[i].ID]
		rightScore := scores[ranked[j].ID]
		if leftScore == rightScore {
			return ranked[i].FullName < ranked[j].FullName
		}
		return leftScore > rightScore
	})

	total := len(ranked)
	efficiencyByOfficer := map[int]Efficiency{}
	for _, officer := range ranked {
		rank := 1
		for _, other := range ranked {
			if scores[other.ID] > scores[officer.ID] {
				rank++
			}
		}
		efficiencyByOfficer[officer.ID] = Efficiency{
			Rank:  rank,
			Total: total,
			Score: round2(scores[officer.ID]),
		}
	}

	result := make([]OfficerWithEfficiency, 0, len(officers))
	for _, officer := range officers {
		result = append(result, OfficerWithEfficiency{
			User:       officer,
			Efficiency: efficiencyByOfficer[officer.ID],
		})
	}
	return result, efficiencyByOfficer
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
