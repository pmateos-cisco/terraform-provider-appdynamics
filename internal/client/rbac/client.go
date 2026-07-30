package rbac

import (
	"context"
	"errors"
	"net/http"
	"strings"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
)

// contentType is the versioned content type the RBAC API requires on every
// request (both Content-Type and Accept) -- unlike every other API in this
// provider, plain application/json is rejected outright with a 415/406
// (verified live).
const contentType = "application/vnd.appd.cntrl+json;v=1"

const basePath = "/controller/api/rbac/v1"

// do is a thin wrapper around the shared alertandrespond.Client (same
// authenticated Controller connection every other resource in this provider
// uses) that always sends the RBAC-required content type.
func do(ctx context.Context, c *client.Client, method, path string, body, out any) error {
	return c.DoTyped(ctx, method, path, contentType, body, out)
}

// IsNotFound reports whether err indicates the RBAC entity doesn't exist.
// Unlike the Alert and Respond API, RBAC returns a bare 500 with a raw Java
// null-pointer-exception message for a GET on a nonexistent ID rather than a
// clean 404 (verified live, e.g. `Cannot invoke "...User.getId()" because
// "user" is null`) -- so this checks for that pattern too, in addition to an
// actual 404 should the API ever return one.
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
