package database

import (
	"context"
	"fmt"
	"net/http"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
)

// Server is a monitored database server as discovered by a collector.
// Read-only: there is no create/update/delete API for these (only GET list
// and GET by ID are documented).
type Server struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	Role         string  `json:"role"`
	Host         string  `json:"host"`
	Port         int     `json:"port"`
	IPAddress    string  `json:"ipAddress"`
	NodeID       int64   `json:"nodeId"`
	ConfigID     int64   `json:"configId"`
	InternalName string  `json:"internalName"`
	BackendIDs   []int64 `json:"backendIds"`
}

func serversPath() string        { return "/controller/rest/databases/servers" }
func serverPath(id int64) string { return fmt.Sprintf("%s/%d", serversPath(), id) }

// ListServers returns every monitored database server in the account.
func ListServers(ctx context.Context, c *client.Client) ([]Server, error) {
	var out []Server
	if err := do(ctx, c, http.MethodGet, serversPath(), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetServer retrieves a single monitored database server by ID.
func GetServer(ctx context.Context, c *client.Client, id int64) (*Server, error) {
	var out Server
	if err := do(ctx, c, http.MethodGet, serverPath(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
