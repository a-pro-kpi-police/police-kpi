package dashboard

import (
	"math"
	"sort"
)

const (
	StatusConfirmed = "confirmed"
	StatusPending   = "pending"
	StatusRejected  = "rejected"

	missingLabel = "—"
)

type Filter struct {
	Month      string
	OfficerID  *int
	DistrictID *int
}

type Entry struct {
	Month            string
	OfficerID        int
	Quantity         float64
	PointsPerUnit    float64
	RawScore         float64
	FinalScore       float64
	Status           string
	OfficerName      string
	OfficerRank      string
	OfficerDistrict  string
	CatalogCode      string
	CatalogIndicator string
	CatalogBlock     string
}

type Officer struct {
	ID           int
	FullName     string
	Rank         string
	DistrictName string
}

type Report struct {
	TotalFinal                float64
	ConfirmedCount            int
	PendingCount              int
	RejectedCount             int
	OfficersCount             int
	ConfirmRate               float64
	ConfirmRateConfirmedCount int
	ConfirmRateDecidedCount   int
	Officers                  []OfficerRow
	Blocks                    []BlockRow
	Trend                     []TrendRow
	StatusTrend               []StatusTrendRow
	IndicatorVolume           []IndicatorVolumeRow
	DistrictReview            []DistrictReviewRow
}

type OfficerRow struct {
	OfficerID int
	Name      string
	Rank      string
	District  string
	Final     float64
	Raw       float64
	Confirmed int
	Pending   int
	Blocks    map[string]float64
}

type BlockRow struct {
	Block string
	Final float64
}

type TrendRow struct {
	Month string
	Final float64
}

type StatusTrendRow struct {
	Month     string
	Confirmed int
	Pending   int
	Rejected  int
}

type IndicatorVolumeRow struct {
	Code      string
	Indicator string
	Quantity  float64
}

type DistrictReviewRow struct {
	District string
	Pending  int
	Rejected int
}

type Policy struct{}

func NewPolicy() Policy {
	return Policy{}
}

func (Policy) BuildReport(entries []Entry, officers []Officer, catalogBlocks []string, filter Filter) Report {
	var activeEntries []Entry
	var confirmed []Entry
	var pending []Entry
	var rejected []Entry
	for _, entry := range entries {
		if entry.Quantity <= 0 {
			continue
		}
		activeEntries = append(activeEntries, entry)
		switch entry.Status {
		case StatusConfirmed:
			confirmed = append(confirmed, entry)
		case StatusPending:
			pending = append(pending, entry)
		case StatusRejected:
			rejected = append(rejected, entry)
		}
	}

	totalFinal := 0.0
	for _, entry := range confirmed {
		totalFinal += entry.FinalScore
	}

	officerRows := buildOfficerRows(activeEntries, officers, filter)
	blockRows := buildBlockRows(confirmed, catalogBlocks)
	trendRows := buildTrendRows(confirmed)
	statusTrendRows := buildStatusTrendRows(activeEntries)
	indicatorRows := buildIndicatorVolumeRows(activeEntries)
	districtReviewRows := buildDistrictReviewRows(pending, rejected)

	rateConfirmed := 0
	rateRejected := 0
	for _, entry := range confirmed {
		if entry.PointsPerUnit >= 0 {
			rateConfirmed++
		}
	}
	for _, entry := range rejected {
		if entry.PointsPerUnit >= 0 {
			rateRejected++
		}
	}
	totalDecided := rateConfirmed + rateRejected
	confirmRate := 0.0
	if totalDecided > 0 {
		confirmRate = round1(100 * float64(rateConfirmed) / float64(totalDecided))
	}

	return Report{
		TotalFinal:                round2(totalFinal),
		ConfirmedCount:            len(confirmed),
		PendingCount:              len(pending),
		RejectedCount:             len(rejected),
		OfficersCount:             len(officerRows),
		ConfirmRate:               confirmRate,
		ConfirmRateConfirmedCount: rateConfirmed,
		ConfirmRateDecidedCount:   totalDecided,
		Officers:                  officerRows,
		Blocks:                    blockRows,
		Trend:                     trendRows,
		StatusTrend:               statusTrendRows,
		IndicatorVolume:           indicatorRows,
		DistrictReview:            districtReviewRows,
	}
}

type officerAggregate struct {
	Name      string
	Rank      string
	District  string
	Final     float64
	Raw       float64
	Confirmed int
	Pending   int
	Blocks    map[string]float64
}

func buildOfficerRows(entries []Entry, officers []Officer, filter Filter) []OfficerRow {
	byOfficer := map[int]*officerAggregate{}
	for _, officer := range officers {
		if filter.OfficerID != nil && officer.ID != *filter.OfficerID {
			continue
		}
		byOfficer[officer.ID] = &officerAggregate{
			Name:     officer.FullName,
			Rank:     officer.Rank,
			District: officer.DistrictName,
			Blocks:   map[string]float64{},
		}
	}
	for _, entry := range entries {
		officer := byOfficer[entry.OfficerID]
		if officer == nil {
			officer = &officerAggregate{Blocks: map[string]float64{}}
			byOfficer[entry.OfficerID] = officer
		}
		if entry.OfficerName != "" {
			officer.Name = entry.OfficerName
		}
		if entry.OfficerRank != "" {
			officer.Rank = entry.OfficerRank
		}
		if entry.OfficerDistrict != "" {
			officer.District = entry.OfficerDistrict
		}
		switch entry.Status {
		case StatusConfirmed:
			officer.Final += entry.FinalScore
			officer.Raw += entry.RawScore
			officer.Confirmed++
			officer.Blocks[label(entry.CatalogBlock)] += entry.FinalScore
		case StatusPending:
			officer.Pending++
		}
	}
	rows := make([]OfficerRow, 0, len(byOfficer))
	for id, aggregate := range byOfficer {
		blockValues := map[string]float64{}
		for block, value := range aggregate.Blocks {
			blockValues[block] = round2(value)
		}
		rows = append(rows, OfficerRow{
			OfficerID: id,
			Name:      aggregate.Name,
			Rank:      aggregate.Rank,
			District:  aggregate.District,
			Final:     round2(aggregate.Final),
			Raw:       round2(aggregate.Raw),
			Confirmed: aggregate.Confirmed,
			Pending:   aggregate.Pending,
			Blocks:    blockValues,
		})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i].Final > rows[j].Final
	})
	return rows
}

func buildBlockRows(confirmed []Entry, catalogBlocks []string) []BlockRow {
	byBlock := map[string]float64{}
	for _, block := range catalogBlocks {
		byBlock[block] = 0
	}
	for _, entry := range confirmed {
		byBlock[label(entry.CatalogBlock)] += entry.FinalScore
	}
	rows := make([]BlockRow, 0, len(catalogBlocks))
	for _, block := range catalogBlocks {
		rows = append(rows, BlockRow{Block: block, Final: round2(byBlock[block])})
	}
	return rows
}

func buildTrendRows(confirmed []Entry) []TrendRow {
	byMonth := map[string]float64{}
	for _, entry := range confirmed {
		byMonth[entry.Month] += entry.FinalScore
	}
	months := sortedKeys(byMonth)
	rows := make([]TrendRow, 0, len(months))
	for _, month := range months {
		rows = append(rows, TrendRow{Month: month, Final: round2(byMonth[month])})
	}
	return rows
}

func buildStatusTrendRows(entries []Entry) []StatusTrendRow {
	byMonth := map[string]StatusTrendRow{}
	for _, entry := range entries {
		counts := byMonth[entry.Month]
		counts.Month = entry.Month
		switch entry.Status {
		case StatusConfirmed:
			counts.Confirmed++
		case StatusPending:
			counts.Pending++
		case StatusRejected:
			counts.Rejected++
		}
		byMonth[entry.Month] = counts
	}
	months := make([]string, 0, len(byMonth))
	for month := range byMonth {
		months = append(months, month)
	}
	sort.Strings(months)
	rows := make([]StatusTrendRow, 0, len(months))
	for _, month := range months {
		rows = append(rows, byMonth[month])
	}
	return rows
}

func buildIndicatorVolumeRows(entries []Entry) []IndicatorVolumeRow {
	byCode := map[string]*IndicatorVolumeRow{}
	for _, entry := range entries {
		code := label(entry.CatalogCode)
		item := byCode[code]
		if item == nil {
			item = &IndicatorVolumeRow{Code: code, Indicator: label(entry.CatalogIndicator)}
			byCode[code] = item
		}
		item.Quantity += entry.Quantity
	}
	rows := make([]IndicatorVolumeRow, 0, len(byCode))
	for _, item := range byCode {
		rows = append(rows, IndicatorVolumeRow{
			Code:      item.Code,
			Indicator: item.Indicator,
			Quantity:  round2(item.Quantity),
		})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i].Quantity > rows[j].Quantity
	})
	return rows
}

func buildDistrictReviewRows(pending []Entry, rejected []Entry) []DistrictReviewRow {
	type counts struct {
		pending  int
		rejected int
	}
	byDistrict := map[string]counts{}
	for _, entry := range append(pending, rejected...) {
		district := label(entry.OfficerDistrict)
		item := byDistrict[district]
		if entry.Status == StatusPending {
			item.pending++
		}
		if entry.Status == StatusRejected {
			item.rejected++
		}
		byDistrict[district] = item
	}
	districts := make([]string, 0, len(byDistrict))
	for district := range byDistrict {
		districts = append(districts, district)
	}
	sort.SliceStable(districts, func(i, j int) bool {
		left := byDistrict[districts[i]]
		right := byDistrict[districts[j]]
		return left.pending+left.rejected > right.pending+right.rejected
	})
	rows := make([]DistrictReviewRow, 0, len(districts))
	for _, district := range districts {
		item := byDistrict[district]
		rows = append(rows, DistrictReviewRow{
			District: district,
			Pending:  item.pending,
			Rejected: item.rejected,
		})
	}
	return rows
}

func sortedKeys(values map[string]float64) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func label(value string) string {
	if value == "" {
		return missingLabel
	}
	return value
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func round1(value float64) float64 {
	return math.Round(value*10) / 10
}
