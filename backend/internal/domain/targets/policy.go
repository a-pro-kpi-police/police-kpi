package targets

import (
	"errors"
	"math"
	"sort"
	"strconv"
)

const (
	StatusBehind   = "behind"
	StatusNoTarget = "no_target"
	StatusOnTarget = "on_target"
)

var (
	ErrCatalogNotFound  = errors.New("target catalog not found")
	ErrDistrictNotFound = errors.New("district not found")
	ErrEmptyGroup       = errors.New("target group has no officers")
	ErrInvalidMonth     = errors.New("invalid month")
	ErrNegativeQuantity = errors.New("target quantity cannot be negative")
	ErrOfficerNotFound  = errors.New("officer not found")
)

type SaveItem struct {
	CatalogID      int
	TargetQuantity float64
}

type Target struct {
	CatalogID      int
	Code           string
	Indicator      string
	Block          string
	Unit           string
	PointsPerUnit  float64
	MonthlyLimit   float64
	IsMass         bool
	IsCritical     bool
	TargetQuantity float64
	TargetScore    float64
}

type Actual struct {
	Quantity        float64
	CountedQuantity float64
	Score           float64
}

type ProgressItem struct {
	Target
	ConfirmedQuantity  float64
	CountedQuantity    float64
	RemainingQuantity  float64
	ConfirmedScore     float64
	AchievementPercent float64
}

type Progress struct {
	Month              string
	OfficerID          int
	ActualScore        float64
	TargetScore        float64
	AchievementPercent float64
	Items              []ProgressItem
}

type Penalty struct {
	Score float64
	Count int
}

type OverviewInput struct {
	OfficerID int
	Name      string
	Progress  Progress
	Penalty   Penalty
}

type OfficerOverview struct {
	OfficerID          int
	Name               string
	ActualScore        float64
	TargetScore        float64
	AchievementPercent float64
	Gap                float64
	Status             string
	PenaltyScore       float64
	PenaltyCount       int
}

type BlockProgress struct {
	Block              string
	ActualScore        float64
	TargetScore        float64
	AchievementPercent float64
}

type Overview struct {
	Month              string
	ActualScore        float64
	TargetScore        float64
	AchievementPercent float64
	OnTargetCount      int
	OfficersCount      int
	PenaltyScore       float64
	PenaltyCount       int
	Officers           []OfficerOverview
	Blocks             []BlockProgress
	Items              []ProgressItem
}

type Policy struct{}

func NewPolicy() Policy {
	return Policy{}
}

func (Policy) ValidateMonth(month string) error {
	if len(month) != 7 || month[4] != '-' {
		return ErrInvalidMonth
	}
	year, errYear := strconv.Atoi(month[:4])
	monthNum, errMonth := strconv.Atoi(month[5:])
	if errYear != nil || errMonth != nil || year < 1 || monthNum < 1 || monthNum > 12 {
		return ErrInvalidMonth
	}
	return nil
}

func (Policy) TargetQuantities(items []SaveItem, validCatalogIDs map[int]bool) (map[int]float64, error) {
	quantities := map[int]float64{}
	for _, item := range items {
		if !validCatalogIDs[item.CatalogID] {
			return nil, ErrCatalogNotFound
		}
		if item.TargetQuantity < 0 {
			return nil, ErrNegativeQuantity
		}
		quantities[item.CatalogID] = item.TargetQuantity
	}
	return quantities, nil
}

func (Policy) AchievementPercent(actualScore, targetScore float64) float64 {
	if targetScore == 0 {
		return 0
	}
	return round1(maxFloat(0, 100*actualScore/targetScore))
}

func (p Policy) BuildProgress(month string, officerID int, monthlyNorm float64, targets []Target, actualByCatalog map[int]Actual) Progress {
	items := make([]ProgressItem, 0, len(targets))
	actualTotal := 0.0
	for _, target := range targets {
		actual := actualByCatalog[target.CatalogID]
		confirmedQuantity := round2(actual.Quantity)
		countedQuantity := round2(actual.CountedQuantity)
		confirmedScore := round2(actual.Score)
		item := ProgressItem{
			Target:             target,
			ConfirmedQuantity:  confirmedQuantity,
			CountedQuantity:    countedQuantity,
			RemainingQuantity:  round2(maxFloat(target.TargetQuantity-confirmedQuantity, 0)),
			ConfirmedScore:     confirmedScore,
			AchievementPercent: p.AchievementPercent(confirmedScore, target.TargetScore),
		}
		items = append(items, item)
		actualTotal += confirmedScore
	}

	actualTotal = round2(actualTotal)
	targetTotal := round2(monthlyNorm)
	if targetTotal == 0 {
		targetTotal = 100
	}
	return Progress{
		Month:              month,
		OfficerID:          officerID,
		ActualScore:        actualTotal,
		TargetScore:        targetTotal,
		AchievementPercent: p.AchievementPercent(actualTotal, targetTotal),
		Items:              items,
	}
}

func (p Policy) BuildOverview(month string, inputs []OverviewInput, includeItems bool) Overview {
	officers := make([]OfficerOverview, 0, len(inputs))
	var items []ProgressItem
	actualTotal := 0.0
	targetTotal := 0.0
	penaltyTotal := 0.0
	penaltyCount := 0
	onTargetCount := 0
	byBlock := map[string]struct {
		ActualScore float64
		TargetScore float64
	}{}

	for _, input := range inputs {
		progress := input.Progress
		penalty := input.Penalty
		gap := round2(progress.ActualScore - progress.TargetScore)
		status := StatusBehind
		if progress.TargetScore == 0 {
			status = StatusNoTarget
		} else if progress.ActualScore >= progress.TargetScore {
			status = StatusOnTarget
			onTargetCount++
		}

		officers = append(officers, OfficerOverview{
			OfficerID:          input.OfficerID,
			Name:               input.Name,
			ActualScore:        progress.ActualScore,
			TargetScore:        progress.TargetScore,
			AchievementPercent: progress.AchievementPercent,
			Gap:                gap,
			Status:             status,
			PenaltyScore:       round2(penalty.Score),
			PenaltyCount:       penalty.Count,
		})
		actualTotal += progress.ActualScore
		targetTotal += progress.TargetScore
		penaltyTotal += penalty.Score
		penaltyCount += penalty.Count
		if includeItems {
			items = progress.Items
		}
		for _, item := range progress.Items {
			block := byBlock[item.Block]
			block.ActualScore += item.ConfirmedScore
			block.TargetScore += item.TargetScore
			byBlock[item.Block] = block
		}
	}

	if !includeItems {
		sortOfficerOverview(officers)
	}

	blockNames := make([]string, 0, len(byBlock))
	for block, values := range byBlock {
		if values.TargetScore > 0 {
			blockNames = append(blockNames, block)
		}
	}
	sort.Strings(blockNames)
	blocks := make([]BlockProgress, 0, len(blockNames))
	for _, blockName := range blockNames {
		values := byBlock[blockName]
		blocks = append(blocks, BlockProgress{
			Block:              blockName,
			ActualScore:        round2(values.ActualScore),
			TargetScore:        round2(values.TargetScore),
			AchievementPercent: p.AchievementPercent(values.ActualScore, values.TargetScore),
		})
	}

	return Overview{
		Month:              month,
		ActualScore:        round2(actualTotal),
		TargetScore:        round2(targetTotal),
		AchievementPercent: p.AchievementPercent(actualTotal, targetTotal),
		OnTargetCount:      onTargetCount,
		OfficersCount:      len(officers),
		PenaltyScore:       round2(penaltyTotal),
		PenaltyCount:       penaltyCount,
		Officers:           officers,
		Blocks:             blocks,
		Items:              items,
	}
}

func sortOfficerOverview(officers []OfficerOverview) {
	statusOrder := map[string]int{StatusBehind: 0, StatusNoTarget: 1, StatusOnTarget: 2}
	sort.SliceStable(officers, func(i, j int) bool {
		left, right := officers[i], officers[j]
		if statusOrder[left.Status] != statusOrder[right.Status] {
			return statusOrder[left.Status] < statusOrder[right.Status]
		}
		leftAchievement := left.AchievementPercent
		rightAchievement := right.AchievementPercent
		if left.TargetScore == 0 {
			leftAchievement = 9999
		}
		if right.TargetScore == 0 {
			rightAchievement = 9999
		}
		if leftAchievement != rightAchievement {
			return leftAchievement < rightAchievement
		}
		if left.Gap != right.Gap {
			return left.Gap < right.Gap
		}
		return left.Name < right.Name
	})
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func round1(value float64) float64 {
	return math.Round(value*10) / 10
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
