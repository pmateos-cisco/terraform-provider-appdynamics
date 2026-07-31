package database

import (
	"context"
	"errors"
	"net/http"
	"strings"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
)

func do(ctx context.Context, c *client.Client, method, path string, body, out any) error {
	return c.DoTyped(ctx, method, path, "application/json", body, out)
}

// IsNotFound reports whether err indicates a missing collector/server:
// either a standard 404, or the 500-with-Java-NPE-message pattern verified
// live for GET on a deleted collector -- the same "not found" quirk as the
// RBAC API.
func IsNotFound(err error) bool {
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	if apiErr.StatusCode == http.StatusNotFound {
		return true
	}
	return apiErr.StatusCode == http.StatusInternalServerError &&
		strings.Contains(apiErr.Body, "is null")
}
