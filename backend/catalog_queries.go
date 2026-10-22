package main

import domaincatalog "kpi-police/backend/internal/domain/catalog"

func catalogItems(items []domaincatalog.Item) []CatalogItem {
	result := make([]CatalogItem, 0, len(items))
	for _, item := range items {
		result = append(result, catalogItem(item))
	}
	return result
}

func catalogItem(item domaincatalog.Item) CatalogItem {
	return CatalogItem{
		ID:            item.ID,
		Code:          item.Code,
		Indicator:     item.Indicator,
		Block:         item.Block,
		Subblock:      item.Subblock,
		NPA:           item.NPA,
		PointsPerUnit: item.PointsPerUnit,
		Unit:          item.Unit,
		SourceURL:     item.SourceURL,
		MonthlyLimit:  item.MonthlyLimit,
		IsMass:        item.IsMass,
		IsCritical:    item.IsCritical,
		IsActive:      item.IsActive,
	}
}
