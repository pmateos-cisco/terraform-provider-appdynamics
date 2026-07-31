package database

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
)

// eventsPath uses the account-wide "_dbmon" pseudo-application, same
// convention as nodesPath.
func eventsPath() string { return "/controller/rest/applications/_dbmon/events" }

// ListEvents queries Database Monitoring agent events within a time range
// (params: time-range-type, duration-in-mins or start-time/end-time,
// event-types, severities -- event-types and severities are both required,
// verified live). Reuses client.Event (from the alertandrespond package)
// since the response shape is identical to the Alert-and-Respond Events API
// (verified live). output=JSON is added automatically (verified live:
// required, same as ListNodes).
func ListEvents(ctx context.Context, c *client.Client, params url.Values) ([]client.Event, error) {
	p := cloneValues(params)
	p.Set("output", "JSON")
	var out []client.Event
	if err := do(ctx, c, http.MethodGet, eventsPath()+"?"+p.Encode(), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func healthRuleViolationsPath(serverID int64) string {
	return fmt.Sprintf("/controller/rest/databases/servers/healthrule-violations/%d", serverID)
}

// ListHealthRuleViolations queries health rule violations for a specific
// monitored database server within a time range (params: time-range-type,
// duration-in-mins or start-time/end-time). Reuses
// client.HealthRuleViolation for the same reason as ListEvents.
func ListHealthRuleViolations(ctx context.Context, c *client.Client, serverID int64, params url.Values) ([]client.HealthRuleViolation, error) {
	p := cloneValues(params)
	p.Set("output", "JSON")
	var out []client.HealthRuleViolation
	if err := do(ctx, c, http.MethodGet, healthRuleViolationsPath(serverID)+"?"+p.Encode(), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func cloneValues(v url.Values) url.Values {
	out := make(url.Values, len(v))
	for k, vals := range v {
		out[k] = append([]string(nil), vals...)
	}
	return out
}
