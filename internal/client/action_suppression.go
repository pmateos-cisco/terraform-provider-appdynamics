package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// ActionSuppression represents an AppDynamics action suppression: a schedule
// during which actions are muted for a scope of entities. RecurringSchedule,
// Affects, and HealthRuleScope are passed through as raw JSON since their
// shape is polymorphic (RecurringSchedule branches by scheduleFrequency like
// Schedule.ScheduleConfiguration; Affects branches by affectedInfoType) — see
// examples/resources/appdynamics_action_suppression for the JSON shapes.
type ActionSuppression struct {
	ID                      int64           `json:"id,omitempty"`
	Name                    string          `json:"name"`
	DisableAgentReporting   bool            `json:"disableAgentReporting"`
	SuppressionScheduleType string          `json:"suppressionScheduleType"`
	Timezone                string          `json:"timezone,omitempty"`
	StartTime               string          `json:"startTime,omitempty"`
	EndTime                 string          `json:"endTime,omitempty"`
	RecurringSchedule       json.RawMessage `json:"recurringSchedule,omitempty"`
	Affects                 json.RawMessage `json:"affects"`
	HealthRuleScope         json.RawMessage `json:"healthRuleScope,omitempty"`
}

// ActionSuppressionSummary is the abbreviated representation returned by the
// action-suppressions list endpoint.
type ActionSuppressionSummary struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Timezone  string `json:"timezone"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
}

func actionSuppressionsPath(applicationID int64) string {
	return fmt.Sprintf("/controller/alerting/rest/v1/applications/%d/action-suppressions", applicationID)
}

func actionSuppressionItemPath(applicationID, actionSuppressionID int64) string {
	return fmt.Sprintf("%s/%d", actionSuppressionsPath(applicationID), actionSuppressionID)
}

func actionSuppressionByNamePath(applicationID int64, name string) string {
	return fmt.Sprintf("%s/action-suppression-by-name/?name=%s", actionSuppressionsPath(applicationID), url.QueryEscape(name))
}

func (c *Client) CreateActionSuppression(ctx context.Context, applicationID int64, as *ActionSuppression) (*ActionSuppression, error) {
	var out ActionSuppression
	if err := c.do(ctx, http.MethodPost, actionSuppressionsPath(applicationID), as, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetActionSuppression(ctx context.Context, applicationID, actionSuppressionID int64) (*ActionSuppression, error) {
	var out ActionSuppression
	if err := c.do(ctx, http.MethodGet, actionSuppressionItemPath(applicationID, actionSuppressionID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetActionSuppressionByName(ctx context.Context, applicationID int64, name string) (*ActionSuppression, error) {
	var out ActionSuppression
	if err := c.do(ctx, http.MethodGet, actionSuppressionByNamePath(applicationID, name), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateActionSuppression(ctx context.Context, applicationID, actionSuppressionID int64, as *ActionSuppression) (*ActionSuppression, error) {
	var out ActionSuppression
	if err := c.do(ctx, http.MethodPut, actionSuppressionItemPath(applicationID, actionSuppressionID), as, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteActionSuppression(ctx context.Context, applicationID, actionSuppressionID int64) error {
	return c.do(ctx, http.MethodDelete, actionSuppressionItemPath(applicationID, actionSuppressionID), nil, nil)
}

func (c *Client) ListActionSuppressions(ctx context.Context, applicationID int64) ([]ActionSuppressionSummary, error) {
	var out []ActionSuppressionSummary
	if err := c.do(ctx, http.MethodGet, actionSuppressionsPath(applicationID), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}
