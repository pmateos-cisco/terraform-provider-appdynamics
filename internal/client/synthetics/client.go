package synthetics

import (
	"context"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
)

// basePath is the undocumented but verified-live "restui" path the Controller
// UI itself uses for synthetic job management via OAuth. The officially
// documented API (POST/GET/PUT against a separate EUM api_server_URL, using
// Basic auth with an EUM account username + license key) requires a second
// credential set this provider doesn't have configured, so this package
// reuses the existing OAuth client against the Controller URL instead --
// verified live to work for list/create/update/delete. Standard
// "application/json" is accepted (unlike the rbac package, which requires a
// versioned content type).
const basePath = "/controller/restui/synthetic"

func do(ctx context.Context, c *client.Client, method, path string, body, out any) error {
	return c.DoTyped(ctx, method, path, "application/json", body, out)
}
