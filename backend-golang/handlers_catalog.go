package main

import (
	"net/http"

	domaincatalog "kpi-police/backend-golang/internal/domain/catalog"
)

func (a *App) handleCatalog(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(a, w, r)
	if !ok {
		return
	}
	var (
		items []domaincatalog.Item
		err   error
	)
	if parseBoolInt(r.URL.Query().Get("positive_only")) {
		items, err = a.catalogRepository().ListActivePositive(r.Context())
	} else {
		items, err = a.catalogRepository().ListActive(r.Context(), user.Role == roleHead)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, catalogItems(items))
}

func (a *App) handleDistricts(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireUser(a, w, r); !ok {
		return
	}
	districts, err := a.userQueryService().ListDistricts(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, districtModels(districts))
}

func (a *App) handleOfficers(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireHead(a, w, r); !ok {
		return
	}
	officers, err := a.userQueryService().ListOfficersWithEfficiency(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	result := make([]OfficerOut, 0, len(officers))
	for _, officer := range officers {
		user := userModel(officer.User)
		info := officer.Efficiency
		result = append(result, OfficerOut{
			ID:              user.ID,
			Username:        user.Username,
			FullName:        user.FullName,
			Rank:            user.Rank,
			Role:            user.Role,
			MonthlyNorm:     round2(user.MonthlyNorm),
			District:        user.District,
			EfficiencyRank:  info.Rank,
			EfficiencyTotal: info.Total,
			EfficiencyScore: info.Score,
		})
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *App) handleMonths(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(a, w, r)
	if !ok {
		return
	}
	months, err := a.userQueryService().ListMonths(r.Context(), domainUser(user))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, months)
}
