package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Policy represents an AppDynamics alert policy. Actions, Events and
// SelectedEntities are passed through as raw JSON since their shapes are
// polymorphic (action references carry per-entity-type details, event
// triggers vary by event category, and entity selection varies by entity
// type) — see examples/resources/appdynamics_policy for the JSON shapes to
// send.
type Policy struct {
	ID                    int64           `json:"id,omitempty"`
	Name                  string          `json:"name"`
	Enabled               bool            `json:"enabled"`
	ExecuteActionsInBatch bool            `json:"executeActionsInBatch"`
	Actions               json.RawMessage `json:"actions"`
	Events                json.RawMessage `json:"events"`
	SelectedEntities      json.RawMessage `json:"selectedEntities"`
}

func policiesPath(applicationID int64) string {
	return fmt.Sprintf("/controller/alerting/rest/v1/applications/%d/policies", applicationID)
}

func policyPath(applicationID, policyID int64) string {
	return fmt.Sprintf("%s/%d", policiesPath(applicationID), policyID)
}

func (c *Client) CreatePolicy(ctx context.Context, applicationID int64, p *Policy) (*Policy, error) {
	var out Policy
	if err := c.do(ctx, http.MethodPost, policiesPath(applicationID), p, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetPolicy(ctx context.Context, applicationID, policyID int64) (*Policy, error) {
	var out Policy
	if err := c.do(ctx, http.MethodGet, policyPath(applicationID, policyID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdatePolicy(ctx context.Context, applicationID, policyID int64, p *Policy) (*Policy, error) {
	var out Policy
	if err := c.do(ctx, http.MethodPut, policyPath(applicationID, policyID), p, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeletePolicy(ctx context.Context, applicationID, policyID int64) error {
	return c.do(ctx, http.MethodDelete, policyPath(applicationID, policyID), nil, nil)
}

// PolicySummary is the abbreviated representation used from the policies list
// endpoint (id, name, enabled only — the list response actually includes
// actions/events/selectedEntityType too, but those are ignored here for
// consistency with this provider's other list data sources; use GetPolicy
// for a specific policy's full detail).
type PolicySummary struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

func (c *Client) ListPolicies(ctx context.Context, applicationID int64) ([]PolicySummary, error) {
	var out []PolicySummary
	if err := c.do(ctx, http.MethodGet, policiesPath(applicationID), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}
