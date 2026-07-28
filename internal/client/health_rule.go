package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// HealthRule represents an AppDynamics health rule. Affects and EvalCriterias
// are passed through as raw JSON since their shape is polymorphic (they vary
// by affected-entity-type and by eval-criteria-type respectively) — see
// examples/resources/appdynamics_health_rule for the JSON shapes to send.
type HealthRule struct {
	ID                      int64           `json:"id,omitempty"`
	Name                    string          `json:"name"`
	Enabled                 bool            `json:"enabled"`
	UseDataFromLastNMinutes int64           `json:"useDataFromLastNMinutes,omitempty"`
	WaitTimeAfterViolation  int64           `json:"waitTimeAfterViolation,omitempty"`
	ScheduleName            string          `json:"scheduleName,omitempty"`
	Affects                 json.RawMessage `json:"affects"`
	EvalCriterias           json.RawMessage `json:"evalCriterias"`
}

func healthRulesPath(applicationID int64) string {
	return fmt.Sprintf("/controller/alerting/rest/v1/applications/%d/health-rules", applicationID)
}

func healthRulePath(applicationID, healthRuleID int64) string {
	return fmt.Sprintf("%s/%d", healthRulesPath(applicationID), healthRuleID)
}

func (c *Client) CreateHealthRule(ctx context.Context, applicationID int64, hr *HealthRule) (*HealthRule, error) {
	var out HealthRule
	if err := c.do(ctx, http.MethodPost, healthRulesPath(applicationID), hr, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetHealthRule(ctx context.Context, applicationID, healthRuleID int64) (*HealthRule, error) {
	var out HealthRule
	if err := c.do(ctx, http.MethodGet, healthRulePath(applicationID, healthRuleID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// HealthRuleSummary is the abbreviated representation returned by the
// health-rules list endpoint (id, name, enabled, affectedEntityType only —
// use GetHealthRule for a specific rule's full affects/evalCriterias detail).
type HealthRuleSummary struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	Enabled            bool   `json:"enabled"`
	AffectedEntityType string `json:"affectedEntityType"`
}

func (c *Client) ListHealthRules(ctx context.Context, applicationID int64) ([]HealthRuleSummary, error) {
	var out []HealthRuleSummary
	if err := c.do(ctx, http.MethodGet, healthRulesPath(applicationID), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// EnableAllHealthRules enables every health rule defined for applicationID in
// a single call.
func (c *Client) EnableAllHealthRules(ctx context.Context, applicationID int64) error {
	return c.do(ctx, http.MethodPut, healthRulesPath(applicationID)+"/enable", nil, nil)
}

// DisableAllHealthRules disables every health rule defined for applicationID
// in a single call.
func (c *Client) DisableAllHealthRules(ctx context.Context, applicationID int64) error {
	return c.do(ctx, http.MethodPut, healthRulesPath(applicationID)+"/disable", nil, nil)
}

func (c *Client) UpdateHealthRule(ctx context.Context, applicationID, healthRuleID int64, hr *HealthRule) (*HealthRule, error) {
	var out HealthRule
	if err := c.do(ctx, http.MethodPut, healthRulePath(applicationID, healthRuleID), hr, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteHealthRule(ctx context.Context, applicationID, healthRuleID int64) error {
	return c.do(ctx, http.MethodDelete, healthRulePath(applicationID, healthRuleID), nil, nil)
}
