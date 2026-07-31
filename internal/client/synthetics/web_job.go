package synthetics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
)

// WebJob represents an AppDynamics Synthetic Web Monitoring job.
// ScheduleRunConfigs, Script, NetworkProfile, PerformanceCriteria, and
// ComposableConfig are passed through as raw JSON: each is a polymorphic or
// deeply nested block (verified live), matching this provider's existing
// JSON-passthrough approach for similar shapes elsewhere.
//
// Exactly one of URL / Script must be set (verified live via the docs and
// consistent with the "simple check" vs. "scripted check" distinction) --
// enforced by the resource layer, not this client.
type WebJob struct {
	ID                  string          `json:"id,omitempty"`
	Version             int             `json:"version,omitempty"`
	Description         string          `json:"description"`
	AppKey              string          `json:"appKey"`
	URL                 string          `json:"url,omitempty"`
	Script              json.RawMessage `json:"script,omitempty"`
	BrowserCodes        []string        `json:"browserCodes"`
	ChromeVersions      []string        `json:"chromeVersions"`
	LocationCodes       []string        `json:"locationCodes"`
	TimeoutSeconds      int             `json:"timeoutSeconds,omitempty"`
	UserEnabled         bool            `json:"userEnabled"`
	ScheduleRunConfigs  json.RawMessage `json:"scheduleRunConfigs,omitempty"`
	NetworkProfile      json.RawMessage `json:"networkProfile,omitempty"`
	PerformanceCriteria json.RawMessage `json:"performanceCriteria,omitempty"`
	ComposableConfig    json.RawMessage `json:"composableConfig,omitempty"`
}

type webJobListEnvelope struct {
	Schedules []WebJob `json:"schedules"`
}

func webJobsPath(applicationID int64) string {
	return fmt.Sprintf("%s/schedule/%d", basePath, applicationID)
}

func webJobUpdatePath(applicationID int64) string {
	return webJobsPath(applicationID) + "/updateSchedule"
}

func webJobGetAllPath(applicationID int64) string {
	return webJobsPath(applicationID) + "/getSchedules"
}

func webJobDeletePath(applicationID int64) string {
	return webJobsPath(applicationID) + "/deleteSchedules"
}

// CreateWebJob creates a new job and returns its server-assigned ID.
// The create/update endpoint returns 204 with no body (verified live,
// contradicting the docs' documented 200-with-full-object response), so the
// newly created job must be located afterward via ListWebJobs.
func CreateWebJob(ctx context.Context, c *client.Client, applicationID int64, job *WebJob) (string, error) {
	job.ID = ""
	job.Version = 0
	if err := do(ctx, c, http.MethodPost, webJobUpdatePath(applicationID), job, nil); err != nil {
		return "", err
	}

	jobs, err := ListWebJobs(ctx, c, applicationID)
	if err != nil {
		return "", err
	}
	for _, j := range jobs {
		if j.Description == job.Description && j.AppKey == job.AppKey {
			return j.ID, nil
		}
	}
	return "", fmt.Errorf("created job not found in schedule list afterward")
}

// UpdateWebJob updates an existing job in place. job.ID and job.Version must
// be set to the current state's values (verified live: the endpoint returns
// 204 with no body here too).
func UpdateWebJob(ctx context.Context, c *client.Client, applicationID int64, job *WebJob) error {
	return do(ctx, c, http.MethodPost, webJobUpdatePath(applicationID), job, nil)
}

// DeleteWebJob permanently deletes a job. Verified live via the (undocumented)
// deleteSchedules endpoint, which takes a JSON array of job IDs and returns
// 204 with no body.
func DeleteWebJob(ctx context.Context, c *client.Client, applicationID int64, jobID string) error {
	return do(ctx, c, http.MethodPost, webJobDeletePath(applicationID), []string{jobID}, nil)
}

// ListWebJobs returns every synthetic web monitoring job configured under
// applicationID.
func ListWebJobs(ctx context.Context, c *client.Client, applicationID int64) ([]WebJob, error) {
	var out webJobListEnvelope
	if err := do(ctx, c, http.MethodGet, webJobGetAllPath(applicationID), nil, &out); err != nil {
		return nil, err
	}
	return out.Schedules, nil
}

// GetWebJob retrieves a single job by ID. There is no single-item GET
// endpoint (verified live: /getSchedule/{id} 404s), so this filters the full
// list client-side and synthesizes a 404 APIError if not found, for
// consistency with client.IsNotFound.
func GetWebJob(ctx context.Context, c *client.Client, applicationID int64, jobID string) (*WebJob, error) {
	jobs, err := ListWebJobs(ctx, c, applicationID)
	if err != nil {
		return nil, err
	}
	for _, j := range jobs {
		if j.ID == jobID {
			return &j, nil
		}
	}
	return nil, &client.APIError{StatusCode: http.StatusNotFound, Body: "synthetic web job not found"}
}
