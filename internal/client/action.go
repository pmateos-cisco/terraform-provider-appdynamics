package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Action represents an AppDynamics alert action. The fields that vary by
// ActionType (e.g. ThreadDumpAction's numberOfThreadDumps, EmailAction's
// emails) are carried in ExtraFields as a raw JSON object and merged onto the
// same top-level object as Name/ActionType/Notes on the wire, matching how the
// Controller API represents them.
type Action struct {
	ID          int64
	Name        string
	ActionType  string
	Notes       string
	ExtraFields json.RawMessage
}

func (a Action) MarshalJSON() ([]byte, error) {
	merged := map[string]any{}
	if len(a.ExtraFields) > 0 {
		if err := json.Unmarshal(a.ExtraFields, &merged); err != nil {
			return nil, fmt.Errorf("extra_fields must be a JSON object: %w", err)
		}
	}
	merged["name"] = a.Name
	merged["actionType"] = a.ActionType
	if a.Notes != "" {
		merged["notes"] = a.Notes
	}
	if a.ID != 0 {
		merged["id"] = a.ID
	}
	return json.Marshal(merged)
}

func (a *Action) UnmarshalJSON(data []byte) error {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	if id, ok := m["id"].(float64); ok {
		a.ID = int64(id)
	}
	delete(m, "id")

	if name, ok := m["name"].(string); ok {
		a.Name = name
	}
	delete(m, "name")

	if at, ok := m["actionType"].(string); ok {
		a.ActionType = at
	}
	delete(m, "actionType")

	if notes, ok := m["notes"].(string); ok {
		a.Notes = notes
	}
	delete(m, "notes")

	extra, err := json.Marshal(m)
	if err != nil {
		return err
	}
	a.ExtraFields = extra
	return nil
}

func actionsPath(applicationID int64) string {
	return fmt.Sprintf("/controller/alerting/rest/v1/applications/%d/actions", applicationID)
}

// actionItemPath uses the API's singular "/action/{id}" form for get/update/delete.
func actionItemPath(applicationID, actionID int64) string {
	return fmt.Sprintf("/controller/alerting/rest/v1/applications/%d/action/%d", applicationID, actionID)
}

func (c *Client) CreateAction(ctx context.Context, applicationID int64, a *Action) (*Action, error) {
	var out Action
	if err := c.do(ctx, http.MethodPost, actionsPath(applicationID), a, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetAction(ctx context.Context, applicationID, actionID int64) (*Action, error) {
	var out Action
	if err := c.do(ctx, http.MethodGet, actionItemPath(applicationID, actionID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateAction(ctx context.Context, applicationID, actionID int64, a *Action) (*Action, error) {
	var out Action
	if err := c.do(ctx, http.MethodPut, actionItemPath(applicationID, actionID), a, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteAction(ctx context.Context, applicationID, actionID int64) error {
	return c.do(ctx, http.MethodDelete, actionItemPath(applicationID, actionID), nil, nil)
}
