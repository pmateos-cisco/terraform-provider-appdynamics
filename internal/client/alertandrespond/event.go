package alertandrespond

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
)

// EntityRef identifies an entity (application, tier, node, business
// transaction, policy, ...) referenced by an Event or HealthRuleViolation.
type EntityRef struct {
	EntityType string `json:"entityType"`
	Name       string `json:"name"`
	EntityID   int64  `json:"entityId"`
}

// EventProperty is a single custom-event property as echoed back by the API.
type EventProperty struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Event is an AppDynamics event (deployment, custom, or any other type
// returned by the events query endpoint). Events are immutable: there is no
// API support for reading a single event by ID, updating one, or deleting
// one — only creating and querying by time range.
type Event struct {
	ID               int64           `json:"id"`
	Type             string          `json:"type"`
	SubType          string          `json:"subType"`
	Severity         string          `json:"severity"`
	Summary          string          `json:"summary"`
	EventTimeMillis  int64           `json:"eventTime"`
	DeepLinkURL      string          `json:"deepLinkUrl"`
	Properties       []EventProperty `json:"properties"`
	AffectedEntities []EntityRef     `json:"affectedEntities"`
}

// HealthRuleViolation is a single health rule violation/incident as returned
// by the health-rule-violations query endpoint.
type HealthRuleViolation struct {
	ID                        int64      `json:"id"`
	Name                      string     `json:"name"`
	Severity                  string     `json:"severity"`
	IncidentStatus            string     `json:"incidentStatus"`
	Description               string     `json:"description"`
	StartTimeInMillis         int64      `json:"startTimeInMillis"`
	EndTimeInMillis           int64      `json:"endTimeInMillis"`
	DeepLinkURL               string     `json:"deepLinkUrl"`
	AffectedEntityDefinition  *EntityRef `json:"affectedEntityDefinition"`
	TriggeredEntityDefinition *EntityRef `json:"triggeredEntityDefinition"`
}

func eventsPath(applicationID int64) string {
	return fmt.Sprintf("/controller/rest/applications/%d/events", applicationID)
}

func healthRuleViolationsPath(applicationID int64) string {
	return fmt.Sprintf("/controller/rest/applications/%d/problems/healthrule-violations", applicationID)
}

var createdEventIDPattern = regexp.MustCompile(`id:(\d+)`)

// CreateEvent posts a new event (deployment, custom, or otherwise) using the
// given form parameters and returns the created event's ID, parsed out of
// the API's plain-text confirmation response (e.g. "Successfully created the
// event id:1018662960") since this endpoint doesn't return JSON even when
// output=JSON is requested on GET queries.
func (c *Client) CreateEvent(ctx context.Context, applicationID int64, form url.Values) (int64, error) {
	body, err := c.doForm(ctx, http.MethodPost, eventsPath(applicationID), form)
	if err != nil {
		return 0, err
	}
	m := createdEventIDPattern.FindSubmatch(body)
	if m == nil {
		return 0, fmt.Errorf("unexpected response creating event: %s", string(body))
	}
	id, err := strconv.ParseInt(string(m[1]), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing created event id: %w", err)
	}
	return id, nil
}

// ListEvents queries events for applicationID within the time range and
// filters described by params (time-range-type, duration-in-mins or
// start-time/end-time, event-types, severities — see the AppDynamics Events
// API docs). output=JSON is added automatically.
func (c *Client) ListEvents(ctx context.Context, applicationID int64, params url.Values) ([]Event, error) {
	params = cloneValues(params)
	params.Set("output", "JSON")
	var out []Event
	path := eventsPath(applicationID) + "?" + params.Encode()
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListHealthRuleViolations queries health rule violations for applicationID
// within the time range described by params (time-range-type,
// duration-in-mins or start-time/end-time). output=JSON is added
// automatically.
func (c *Client) ListHealthRuleViolations(ctx context.Context, applicationID int64, params url.Values) ([]HealthRuleViolation, error) {
	params = cloneValues(params)
	params.Set("output", "JSON")
	var out []HealthRuleViolation
	path := healthRuleViolationsPath(applicationID) + "?" + params.Encode()
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
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
