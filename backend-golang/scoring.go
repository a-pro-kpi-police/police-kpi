package main

import (
	"context"

	appscoring "kpi-police/backend-golang/internal/application/scoring"
)

type ScoringResult struct {
	OfficerID        int     `json:"officer_id"`
	Month            string  `json:"month"`
	ReportedQuantity float64 `json:"reported_quantity"`
	CountedQuantity  float64 `json:"counted_quantity"`
	PositiveScore    float64 `json:"positive_score"`
	NegativeScore    float64 `json:"negative_score"`
	MassScore        float64 `json:"mass_score"`
	MassShare        float64 `json:"mass_share"`
	AntiCoeff        float64 `json:"anti_coeff"`
	ConditionCoeff   float64 `json:"condition_coeff"`
	Critical         bool    `json:"critical"`
	FinalScore       float64 `json:"final_score"`
}

func recalculateOfficerMonth(ctx context.Context, q dbtx, officerID int, month string) (ScoringResult, error) {
	result, err := scoringService(q).RecalculateOfficerMonth(ctx, officerID, month)
	if err != nil {
		return ScoringResult{}, err
	}
	return scoringResult(result), nil
}

func scoringResult(monthResult appscoring.MonthResult) ScoringResult {
	result := monthResult.Result
	return ScoringResult{
		OfficerID:        monthResult.OfficerID,
		Month:            monthResult.Month,
		ReportedQuantity: result.ReportedQuantity,
		CountedQuantity:  result.CountedQuantity,
		PositiveScore:    result.PositiveScore,
		NegativeScore:    result.NegativeScore,
		MassScore:        result.MassScore,
		MassShare:        result.MassShare,
		AntiCoeff:        result.AntiCoeff,
		ConditionCoeff:   result.ConditionCoeff,
		Critical:         result.Critical,
		FinalScore:       result.FinalScore,
	}
}
