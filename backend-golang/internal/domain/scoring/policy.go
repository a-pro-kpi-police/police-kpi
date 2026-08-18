package scoring

import "math"

const (
	StatusConfirmed = "confirmed"

	massThreshold1   = 0.40
	massThreshold2   = 0.60
	massCoefficient1 = 0.85
	massCoefficient2 = 0.70
)

type CatalogSnapshot struct {
	MonthlyLimit float64
	IsMass       bool
	IsCritical   bool
}

type Entry struct {
	ID              int
	CatalogID       int
	Quantity        float64
	CountedQuantity float64
	PointsPerUnit   float64
	Coeff           float64
	RawScore        float64
	CappedScore     float64
	AntiCoeff       float64
	FinalScore      float64
	Status          string
	Catalog         *CatalogSnapshot
}

type Result struct {
	ReportedQuantity float64
	CountedQuantity  float64
	PositiveScore    float64
	NegativeScore    float64
	MassScore        float64
	MassShare        float64
	AntiCoeff        float64
	ConditionCoeff   float64
	Critical         bool
	FinalScore       float64
}

type Calculation struct {
	Entries []Entry
	Result  Result
}

type Policy struct{}

func NewPolicy() Policy {
	return Policy{}
}

func (Policy) Calculate(entries []Entry) Calculation {
	scored := make([]Entry, len(entries))
	copy(scored, entries)

	var confirmed []*Entry
	byCatalog := map[int][]*Entry{}
	for i := range scored {
		entry := &scored[i]
		entry.RawScore = round2(entry.Quantity * entry.PointsPerUnit)
		if entry.Status == StatusConfirmed {
			confirmed = append(confirmed, entry)
			byCatalog[entry.CatalogID] = append(byCatalog[entry.CatalogID], entry)
			continue
		}
		previewQuantity := math.Min(math.Max(entry.Quantity, 0), entryLimit(entry))
		entry.CountedQuantity = round2(previewQuantity)
		entry.CappedScore = round2(previewQuantity * entry.PointsPerUnit)
		entry.AntiCoeff = 1.0
		entry.FinalScore = round2(entry.CappedScore * entry.Coeff)
	}

	for _, catalogEntries := range byCatalog {
		if len(catalogEntries) == 0 {
			continue
		}
		remaining := entryLimit(catalogEntries[0])
		for _, entry := range catalogEntries {
			quantity := math.Max(entry.Quantity, 0)
			counted := quantity
			if !math.IsInf(remaining, 1) {
				counted = math.Min(quantity, math.Max(remaining, 0))
				remaining -= counted
			}
			entry.CountedQuantity = round2(counted)
			entry.CappedScore = round2(entry.CountedQuantity * entry.PointsPerUnit)
		}
	}

	var positiveScore, negativeScore, massScore float64
	for _, entry := range confirmed {
		if entry.PointsPerUnit >= 0 {
			positiveScore += entry.CappedScore
			if entry.Catalog != nil && entry.Catalog.IsMass {
				massScore += entry.CappedScore
			}
		} else {
			negativeScore += entry.CappedScore
		}
	}
	positiveScore = round2(positiveScore)
	negativeScore = round2(negativeScore)
	massScore = round2(massScore)

	massShare := 0.0
	if positiveScore != 0 {
		massShare = massScore / positiveScore
	}
	anti := massCoefficient(massShare)

	critical := false
	for _, entry := range confirmed {
		if entry.Quantity > 0 && entry.Catalog != nil && entry.Catalog.IsCritical {
			critical = true
			break
		}
	}

	monthCoeff := 1.0
	if len(confirmed) > 0 {
		monthCoeff = confirmed[0].Coeff
	}
	for _, entry := range confirmed {
		entry.Coeff = monthCoeff
		entry.AntiCoeff = anti
	}

	adjustedPositive := positiveScore * anti * monthCoeff
	targetFinal := 0.0
	if !critical {
		targetFinal = round2(math.Max(0, adjustedPositive+negativeScore))
	}
	if critical {
		for _, entry := range confirmed {
			entry.FinalScore = 0
		}
	} else {
		penaltyScale := 1.0
		if negativeScore < 0 && adjustedPositive+negativeScore < 0 {
			if adjustedPositive > 0 {
				penaltyScale = adjustedPositive / math.Abs(negativeScore)
			} else {
				penaltyScale = 0
			}
		}
		var scoredEntries []*Entry
		for _, entry := range confirmed {
			if entry.PointsPerUnit >= 0 {
				entry.FinalScore = round2(entry.CappedScore * anti * monthCoeff)
			} else {
				entry.FinalScore = round2(entry.CappedScore * penaltyScale)
			}
			if entry.CountedQuantity > 0 {
				scoredEntries = append(scoredEntries, entry)
			}
		}
		if len(scoredEntries) > 0 {
			allocated := 0.0
			for _, entry := range confirmed {
				allocated += entry.FinalScore
			}
			adjustment := round2(targetFinal - round2(allocated))
			last := scoredEntries[len(scoredEntries)-1]
			last.FinalScore = round2(last.FinalScore + adjustment)
		}
	}

	reportedQuantity := 0.0
	countedQuantity := 0.0
	for _, entry := range confirmed {
		reportedQuantity += entry.Quantity
		countedQuantity += entry.CountedQuantity
	}

	return Calculation{
		Entries: scored,
		Result: Result{
			ReportedQuantity: round2(reportedQuantity),
			CountedQuantity:  round2(countedQuantity),
			PositiveScore:    positiveScore,
			NegativeScore:    negativeScore,
			MassScore:        massScore,
			MassShare:        round6(massShare),
			AntiCoeff:        anti,
			ConditionCoeff:   monthCoeff,
			Critical:         critical,
			FinalScore:       targetFinal,
		},
	}
}

func massCoefficient(massShare float64) float64 {
	if massShare > massThreshold2 {
		return massCoefficient2
	}
	if massShare > massThreshold1 {
		return massCoefficient1
	}
	return 1.0
}

func entryLimit(entry *Entry) float64 {
	if entry.Catalog != nil && entry.Catalog.MonthlyLimit > 0 {
		return entry.Catalog.MonthlyLimit
	}
	return math.Inf(1)
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func round6(value float64) float64 {
	return math.Round(value*1_000_000) / 1_000_000
}
