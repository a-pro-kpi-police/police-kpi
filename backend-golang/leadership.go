package main

import (
	"context"
	"net/http"
	"strings"

	"kpi-police/backend-golang/internal/infrastructure/viewingscope"
)

func isEntryViewerRole(role string) bool {
	return isEntryManagerRole(role) || role == roleLeadership
}

func requireEntryViewer(a *App, w http.ResponseWriter, r *http.Request) (*User, bool) {
	user, ok := requireUser(a, w, r)
	if !ok {
		return nil, false
	}
	if !isEntryViewerRole(user.Role) {
		writeError(w, http.StatusForbidden, "Недостаточно прав")
		return nil, false
	}
	return user, true
}

func requireHeadViewer(a *App, w http.ResponseWriter, r *http.Request) (*User, bool) {
	user, ok := requireUser(a, w, r)
	if !ok {
		return nil, false
	}
	if user.Role != roleHead && user.Role != roleLeadership {
		writeError(w, http.StatusForbidden, "Недостаточно прав")
		return nil, false
	}
	return user, true
}

// Allow only existing viewing and session routes. New business endpoints are
// denied by default for leadership, regardless of the handler's own policy.
func (a *App) leadershipPermissions(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := a.currentUser(r)
		if err == nil && user.Role == roleLeadership && !leadershipRouteAllowed(r) {
			writeError(w, http.StatusForbidden, "Доступно только для просмотра")
			return
		}
		if err == nil && user.Role == roleLeadership {
			if value := r.URL.Query().Get("district_id"); value != "" {
				id, parseErr := parseInt(value)
				if parseErr != nil || id <= 0 {
					writeError(w, 400, "Рота указана неверно")
					return
				}
				if !user.LeadershipScope.allows(id) {
					writeError(w, 403, "Недостаточно прав для другой роты")
					return
				}
			}
		}
		if err == nil {
			ctx := context.WithValue(r.Context(), currentUserKey{}, user)
			if user.Role == roleLeadership && (user.LeadershipScope == nil || user.LeadershipScope.Mode != "all") {
				ids := []int{}
				if user.LeadershipScope != nil && user.LeadershipScope.Mode == "selected" {
					ids = user.LeadershipScope.DistrictIDs
				}
				ctx = viewingscope.WithDistricts(ctx, ids)
			}
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

func leadershipRouteAllowed(r *http.Request) bool {
	if r.URL.Path == "/api/auth/login" {
		return r.Method == http.MethodPost
	}
	if r.Method != http.MethodGet {
		return false
	}
	switch r.URL.Path {
	case "/api/auth/me", "/api/catalog", "/api/districts", "/api/officers",
		"/api/rating", "/api/months", "/api/entries", "/api/dashboard",
		"/api/targets", "/api/group-targets", "/api/target-progress", "/api/target-overview":
		return true
	}
	return strings.HasPrefix(r.URL.Path, "/api/entries/")
}

type currentUserKey struct{}
