package synthetics

import (
	"context"
	"net/http"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
)

// APIApplication is a Synthetic API Monitoring "application": a lightweight,
// account-wide container that Synthetic API Monitoring jobs are grouped
// under. Unlike the Browser RUM app a Synthetic Web Monitoring job requires
// (which has no discovered create/delete API and must be set up via the
// Controller UI), this container has a real, fully API-manageable lifecycle
// -- verified live.
type APIApplication struct {
	ID      int64  `json:"id,omitempty"`
	Version int    `json:"version,omitempty"`
	Name    string `json:"name"`
	AppKey  string `json:"appKey,omitempty"`
}

const apiApplicationsListPath = "/controller/restui/eumApplications/getEumApiMonitoringApplications"
const apiApplicationsCreatePath = "/controller/restui/allApplications/createApplication?applicationType=SYNTH_API_MONITORING"
const apiApplicationsDeletePath = "/controller/restui/allApplications/deleteApplication"

// CreateAPIApplication creates a new Synthetic API Monitoring application
// container and returns its server-assigned ID. The create response doesn't
// include appKey (verified live: applicationTypeInfo/eumAppName are null
// immediately after create), so the caller should follow up with
// GetAPIApplication to get the full, list-endpoint view including appKey.
func CreateAPIApplication(ctx context.Context, c *client.Client, name string) (int64, error) {
	var out APIApplication
	body := map[string]string{"name": name}
	if err := do(ctx, c, http.MethodPost, apiApplicationsCreatePath, body, &out); err != nil {
		return 0, err
	}
	return out.ID, nil
}

// DeleteAPIApplication permanently deletes an API Monitoring application
// container (and, presumptively, every job under it). Verified live: the
// endpoint takes a bare JSON number (not an array or object -- both were
// rejected with a deserialization error) and returns 204 with no body.
func DeleteAPIApplication(ctx context.Context, c *client.Client, applicationID int64) error {
	return do(ctx, c, http.MethodPost, apiApplicationsDeletePath, applicationID, nil)
}

// ListAPIApplications returns every Synthetic API Monitoring application in
// the account. There is no per-application GET (verified live: no working
// single-item endpoint was found), so GetAPIApplication filters this list.
func ListAPIApplications(ctx context.Context, c *client.Client) ([]APIApplication, error) {
	var out []APIApplication
	if err := do(ctx, c, http.MethodGet, apiApplicationsListPath, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetAPIApplication retrieves a single application by ID, synthesizing a 404
// APIError if not found (for consistency with client.IsNotFound).
func GetAPIApplication(ctx context.Context, c *client.Client, applicationID int64) (*APIApplication, error) {
	apps, err := ListAPIApplications(ctx, c)
	if err != nil {
		return nil, err
	}
	for _, a := range apps {
		if a.ID == applicationID {
			return &a, nil
		}
	}
	return nil, &client.APIError{StatusCode: http.StatusNotFound, Body: "synthetic API monitoring application not found"}
}
