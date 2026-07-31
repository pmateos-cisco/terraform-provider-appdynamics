package database

import (
	"context"
	"net/http"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
)

// Node is a Database Monitoring application node (a DB Agent instance), as
// returned by the account-wide "_dbmon" pseudo-application's nodes
// endpoint. Read-only.
type Node struct {
	ID                  int64  `json:"id"`
	Name                string `json:"name"`
	Type                string `json:"type"`
	TierID              int64  `json:"tierId"`
	TierName            string `json:"tierName"`
	MachineID           int64  `json:"machineId"`
	MachineName         string `json:"machineName"`
	MachineOSType       string `json:"machineOSType"`
	AgentType           string `json:"agentType"`
	AppAgentVersion     string `json:"appAgentVersion"`
	MachineAgentVersion string `json:"machineAgentVersion"`
}

// nodesPath uses the account-wide "_dbmon" pseudo-application, the same
// convention the Controller uses for Database Monitoring nodes/events
// rather than a real business application ID.
func nodesPath() string { return "/controller/rest/applications/_dbmon/nodes" }

// ListNodes returns every Database Monitoring node in the account.
// output=JSON is required (verified live: without it, this endpoint
// returns XML regardless of the Accept header).
func ListNodes(ctx context.Context, c *client.Client) ([]Node, error) {
	var out []Node
	if err := do(ctx, c, http.MethodGet, nodesPath()+"?output=JSON", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}
