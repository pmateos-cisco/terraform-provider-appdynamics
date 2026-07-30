package alertandrespond

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// EmailDigest represents an AppDynamics email digest: a periodic rollup
// email binding trigger events on a set of entities to a set of actions.
// Actions, Events and SelectedEntities are passed through as raw JSON since
// their shapes are polymorphic, same as Policy. Unlike Policy, email digests
// do not support executeActionsInBatch -- the API rejects it as invalid for
// this resource type (verified live), so there's no corresponding field
// here.
type EmailDigest struct {
	ID               int64           `json:"id,omitempty"`
	Name             string          `json:"name"`
	Enabled          bool            `json:"enabled"`
	Frequency        int64           `json:"frequency"`
	Actions          json.RawMessage `json:"actions"`
	Events           json.RawMessage `json:"events"`
	SelectedEntities json.RawMessage `json:"selectedEntities"`
}

// EmailDigestSummary is the abbreviated representation returned by the
// email-digests list endpoint (id, name, enabled only -- use GetEmailDigest
// for a specific digest's full detail).
type EmailDigestSummary struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

func emailDigestsPath(applicationID int64) string {
	return fmt.Sprintf("/controller/alerting/rest/v1/applications/%d/email-digests", applicationID)
}

func emailDigestPath(applicationID, emailDigestID int64) string {
	return fmt.Sprintf("%s/%d", emailDigestsPath(applicationID), emailDigestID)
}

func (c *Client) CreateEmailDigest(ctx context.Context, applicationID int64, ed *EmailDigest) (*EmailDigest, error) {
	var out EmailDigest
	if err := c.do(ctx, http.MethodPost, emailDigestsPath(applicationID), ed, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetEmailDigest(ctx context.Context, applicationID, emailDigestID int64) (*EmailDigest, error) {
	var out EmailDigest
	if err := c.do(ctx, http.MethodGet, emailDigestPath(applicationID, emailDigestID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateEmailDigest(ctx context.Context, applicationID, emailDigestID int64, ed *EmailDigest) (*EmailDigest, error) {
	var out EmailDigest
	if err := c.do(ctx, http.MethodPut, emailDigestPath(applicationID, emailDigestID), ed, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteEmailDigest(ctx context.Context, applicationID, emailDigestID int64) error {
	return c.do(ctx, http.MethodDelete, emailDigestPath(applicationID, emailDigestID), nil, nil)
}

func (c *Client) ListEmailDigests(ctx context.Context, applicationID int64) ([]EmailDigestSummary, error) {
	var out []EmailDigestSummary
	if err := c.do(ctx, http.MethodGet, emailDigestsPath(applicationID), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}
