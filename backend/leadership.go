package main

import (
	"net/http"
	"strings"
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

func leadershipAverageScope(user *User) int {
	if user.Role == roleLeadership && user.DistrictID.Valid {
		return int(user.DistrictID.Int64)
	}
	return 0
}
