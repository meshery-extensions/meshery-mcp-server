package meshery

import (
	"net/http"
	"strings"
)

// RouteFamily groups Cloud routes that share a per-user rate budget.
type RouteFamily string

const (
	FamilyClaims       RouteFamily = "claims"
	FamilySchedules    RouteFamily = "schedules"
	FamilyLedgers      RouteFamily = "ledgers"
	FamilyContentItems RouteFamily = "content_items"
	FamilyEventsWrite  RouteFamily = "events_write"
	FamilyEventsRead   RouteFamily = "events_read"
	FamilyOther        RouteFamily = "other"
)

// FamilyForRoute maps a Cloud request to its route family. Unknown routes
// are not client-side rate limited (Cloud still enforces its own limits).
func FamilyForRoute(method, path string) RouteFamily {
	p := strings.ToLower(path)
	switch {
	case strings.Contains(p, "/claims"):
		return FamilyClaims
	case strings.Contains(p, "/schedules"):
		return FamilySchedules
	case strings.Contains(p, "/ledgers"):
		return FamilyLedgers
	case strings.Contains(p, "/content_items") || strings.Contains(p, "/content-items"):
		return FamilyContentItems
	case strings.Contains(p, "/events"):
		if method == http.MethodGet {
			return FamilyEventsRead
		}
		return FamilyEventsWrite
	}
	return FamilyOther
}
