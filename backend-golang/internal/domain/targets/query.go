package targets

type District struct {
	ID    int
	Name  string
	Coeff float64
}

type Officer struct {
	ID          int
	FullName    string
	MonthlyNorm float64
	District    *District
}

type CatalogItem struct {
	CatalogID     int
	Code          string
	Indicator     string
	Block         string
	Unit          string
	PointsPerUnit float64
	MonthlyLimit  float64
	IsMass        bool
	IsCritical    bool
}

type GroupTarget struct {
	Target Target
	Mixed  bool
}

type OverviewFilter struct {
	Month      string
	OfficerID  *int
	DistrictID *int
}

func (Policy) BuildTargets(catalog []CatalogItem, quantities map[int]float64, coeff float64) []Target {
	result := make([]Target, 0, len(catalog))
	for _, item := range catalog {
		targetQuantity := round2(quantities[item.CatalogID])
		result = append(result, Target{
			CatalogID:      item.CatalogID,
			Code:           item.Code,
			Indicator:      item.Indicator,
			Block:          item.Block,
			Unit:           item.Unit,
			PointsPerUnit:  round2(item.PointsPerUnit),
			MonthlyLimit:   round2(item.MonthlyLimit),
			IsMass:         item.IsMass,
			IsCritical:     item.IsCritical,
			TargetQuantity: targetQuantity,
			TargetScore:    round2(targetQuantity * item.PointsPerUnit * coeff),
		})
	}
	return result
}

func (p Policy) BuildGroupTargets(catalog []CatalogItem, quantities map[int]map[int]float64, officerIDs []int, coeff float64) []GroupTarget {
	result := make([]GroupTarget, 0, len(catalog))
	for _, item := range catalog {
		values := make([]float64, 0, len(officerIDs))
		for _, officerID := range officerIDs {
			values = append(values, round2(quantities[officerID][item.CatalogID]))
		}
		mixed := false
		if len(values) > 1 {
			first := values[0]
			for _, value := range values[1:] {
				if value != first {
					mixed = true
					break
				}
			}
		}
		targetQuantity := 0.0
		if !mixed && len(values) > 0 {
			targetQuantity = values[0]
		}
		result = append(result, GroupTarget{
			Target: p.BuildTargets([]CatalogItem{item}, map[int]float64{item.CatalogID: targetQuantity}, coeff)[0],
			Mixed:  mixed,
		})
	}
	return result
}
