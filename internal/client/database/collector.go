package database

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
)

// Collector represents an AppDynamics Database Visibility collector config.
// Only the fields common across every database type are named; everything
// else -- SSL/CyberArk settings, JDBC properties, custom metrics/events,
// sub-configs, and dozens of other type-specific fields (verified live:
// ~50 of them) -- is captured in Extra and merged back onto the same flat
// JSON object on the wire. This is necessary because the update endpoint
// rejects any request that omits existing fields (verified live: a partial
// update with just id+name was rejected with a validation error, contrary
// to what might be assumed from "id" being the only documented requirement)
// -- Extra preserves whatever the API last returned for those fields so a
// full round-trip never silently drops them.
type Collector struct {
	ID        int64
	Version   int
	Type      string
	Name      string
	Hostname  string
	Port      int
	Username  string
	Password  string
	AgentName string
	Enabled   bool
	Extra     json.RawMessage
}

var collectorKnownFields = []string{"id", "version", "type", "name", "hostname", "port", "username", "password", "agentName", "enabled"}

func (c Collector) MarshalJSON() ([]byte, error) {
	m := map[string]json.RawMessage{}
	if len(c.Extra) > 0 {
		if err := json.Unmarshal(c.Extra, &m); err != nil {
			return nil, fmt.Errorf("unmarshaling collector extra fields: %w", err)
		}
	}
	fields := map[string]any{
		"id": c.ID, "version": c.Version, "type": c.Type, "name": c.Name,
		"hostname": c.Hostname, "port": c.Port, "username": c.Username,
		"password": c.Password, "agentName": c.AgentName, "enabled": c.Enabled,
	}
	for k, v := range fields {
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		m[k] = b
	}
	return json.Marshal(m)
}

func (c *Collector) UnmarshalJSON(data []byte) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	extract := func(key string, dst any) error {
		raw, ok := m[key]
		if !ok || string(raw) == "null" {
			return nil
		}
		return json.Unmarshal(raw, dst)
	}
	if err := extract("id", &c.ID); err != nil {
		return err
	}
	if err := extract("version", &c.Version); err != nil {
		return err
	}
	if err := extract("type", &c.Type); err != nil {
		return err
	}
	if err := extract("name", &c.Name); err != nil {
		return err
	}
	if err := extract("hostname", &c.Hostname); err != nil {
		return err
	}
	if err := extract("port", &c.Port); err != nil {
		return err
	}
	if err := extract("username", &c.Username); err != nil {
		return err
	}
	if err := extract("password", &c.Password); err != nil {
		return err
	}
	if err := extract("agentName", &c.AgentName); err != nil {
		return err
	}
	if err := extract("enabled", &c.Enabled); err != nil {
		return err
	}
	for _, k := range collectorKnownFields {
		delete(m, k)
	}
	extra, err := json.Marshal(m)
	if err != nil {
		return err
	}
	c.Extra = extra
	return nil
}

type collectorListItem struct {
	Config Collector `json:"config"`
}

func collectorsPath() string           { return "/controller/rest/databases/collectors" }
func collectorPath(id int64) string    { return fmt.Sprintf("%s/%d", collectorsPath(), id) }
func collectorCreatePath() string      { return collectorsPath() + "/create" }
func collectorUpdatePath() string      { return collectorsPath() + "/update" }
func collectorBatchDeletePath() string { return collectorsPath() + "/batchDelete" }

// CreateCollector creates a new collector and returns its server-assigned
// ID. The create endpoint returns 201 with no body (verified live), so the
// caller must re-list to find the assigned ID -- matched by name, which the
// API enforces as unique (verified live: every collector's nameUnique field
// is true).
func CreateCollector(ctx context.Context, c *client.Client, col *Collector) (int64, error) {
	col.ID = 0
	col.Version = 0
	if err := do(ctx, c, http.MethodPost, collectorCreatePath(), col, nil); err != nil {
		return 0, err
	}

	cols, err := ListCollectors(ctx, c)
	if err != nil {
		return 0, err
	}
	for _, found := range cols {
		if found.Name == col.Name {
			return found.ID, nil
		}
	}
	return 0, fmt.Errorf("created collector not found in list afterward")
}

// GetCollector retrieves a single collector's full config by ID.
func GetCollector(ctx context.Context, c *client.Client, id int64) (*Collector, error) {
	var out Collector
	if err := do(ctx, c, http.MethodGet, collectorPath(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateCollector updates an existing collector. col must be the complete
// config (verified live: partial updates are rejected), with col.ID set.
// Returns the updated collector, since this endpoint (unlike create) does
// return the full object.
func UpdateCollector(ctx context.Context, c *client.Client, col *Collector) (*Collector, error) {
	var out Collector
	if err := do(ctx, c, http.MethodPost, collectorUpdatePath(), col, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteCollector permanently deletes a single collector by ID.
func DeleteCollector(ctx context.Context, c *client.Client, id int64) error {
	return do(ctx, c, http.MethodDelete, collectorPath(id), nil, nil)
}

// BatchDeleteCollectors permanently deletes multiple collectors in one
// request.
func BatchDeleteCollectors(ctx context.Context, c *client.Client, ids []int64) error {
	return do(ctx, c, http.MethodPost, collectorBatchDeletePath(), ids, nil)
}

// ListCollectors returns every database collector in the account. The list
// endpoint wraps each collector's config in a richer status object
// (collectorStatus, eventSummary, licensesUsed, ...) -- only config is
// extracted here, matching what GetCollector returns for a single item.
func ListCollectors(ctx context.Context, c *client.Client) ([]Collector, error) {
	var out []collectorListItem
	if err := do(ctx, c, http.MethodGet, collectorsPath(), nil, &out); err != nil {
		return nil, err
	}
	cols := make([]Collector, 0, len(out))
	for _, item := range out {
		cols = append(cols, item.Config)
	}
	return cols, nil
}
