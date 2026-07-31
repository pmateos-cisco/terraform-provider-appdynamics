package synthetics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
)

// APIJob represents an AppDynamics Synthetic API Monitoring job. Unlike a
// WebJob, it has no url/browserCodes/chromeVersions choice -- browserCodes is
// always ["API"] internally (verified live) and is set by this package, not
// exposed as a field here, since it's never anything else for this job type.
// APIMetadata, ScheduleRunConfigs, PerformanceCriteria, and ComposableConfig
// are JSON passthrough for the same reasons as WebJob's equivalent fields.
//
// AppKey is deliberately absent: verified live that omitting it entirely
// lets the server auto-assign the key tied to the target applicationID, and
// that reusing another application's key (or an arbitrary string) causes
// create to fail with "API Monitoring job can't be created under this
// application" -- the applicationID in the URL, not an appKey in the body,
// is what actually scopes the job.
type APIJob struct {
	ID                  string          `json:"id,omitempty"`
	Version             int             `json:"version,omitempty"`
	Description         string          `json:"description"`
	APIMetadata         json.RawMessage `json:"apiMetadata"`
	LocationCodes       []string        `json:"locationCodes"`
	TimeoutSeconds      int             `json:"timeoutSeconds,omitempty"`
	UserEnabled         bool            `json:"userEnabled"`
	ScheduleRunConfigs  json.RawMessage `json:"scheduleRunConfigs,omitempty"`
	PerformanceCriteria json.RawMessage `json:"performanceCriteria,omitempty"`
	ComposableConfig    json.RawMessage `json:"composableConfig,omitempty"`

	// BrowserCodes is always ["API"] on the wire; set by CreateAPIJob so
	// callers never need to populate it themselves.
	BrowserCodes []string `json:"browserCodes"`
}

type apiJobListEnvelope struct {
	Schedules []APIJob `json:"schedules"`
}

func apiJobsPath(applicationID int64) string {
	return fmt.Sprintf("%s/api-schedule/%d", basePath, applicationID)
}

func apiJobUpdatePath(applicationID int64) string {
	return apiJobsPath(applicationID) + "/updateSchedule"
}

func apiJobGetAllPath(applicationID int64) string {
	return apiJobsPath(applicationID) + "/getSchedules"
}

func apiJobDeletePath(applicationID int64) string {
	return apiJobsPath(applicationID) + "/deleteSchedules"
}

// CreateAPIJob creates a new job and returns its server-assigned ID. Like
// CreateWebJob, the create/update endpoint returns 204 with no body
// (verified live), so the newly created job must be located afterward via
// ListAPIJobs.
func CreateAPIJob(ctx context.Context, c *client.Client, applicationID int64, job *APIJob) (string, error) {
	job.ID = ""
	job.Version = 0
	job.BrowserCodes = []string{"API"}
	if err := do(ctx, c, http.MethodPost, apiJobUpdatePath(applicationID), job, nil); err != nil {
		return "", err
	}

	jobs, err := ListAPIJobs(ctx, c, applicationID)
	if err != nil {
		return "", err
	}
	for _, j := range jobs {
		if j.Description == job.Description {
			return j.ID, nil
		}
	}
	return "", fmt.Errorf("created job not found in schedule list afterward")
}

// UpdateAPIJob updates an existing job in place. job.ID and job.Version must
// be set to the current state's values.
func UpdateAPIJob(ctx context.Context, c *client.Client, applicationID int64, job *APIJob) error {
	job.BrowserCodes = []string{"API"}
	return do(ctx, c, http.MethodPost, apiJobUpdatePath(applicationID), job, nil)
}

// DeleteAPIJob permanently deletes a job via the (undocumented)
// deleteSchedules endpoint, which takes a JSON array of job IDs and returns
// 204 with no body.
//
// CAUTION (verified live): deleting a job under an application was observed,
// once, to leave that application's getSchedules endpoint permanently
// returning 500 InternalServerException afterward, on an application that
// had been working normally moments before. The application itself was
// separately removed via the Controller UI before root cause could be
// isolated, so it's unconfirmed whether this delete call was the actual
// cause or a coincidental lab-environment issue -- but treat deleting the
// last job under a long-lived application with caution.
func DeleteAPIJob(ctx context.Context, c *client.Client, applicationID int64, jobID string) error {
	return do(ctx, c, http.MethodPost, apiJobDeletePath(applicationID), []string{jobID}, nil)
}

// ListAPIJobs returns every synthetic API monitoring job configured under
// applicationID.
func ListAPIJobs(ctx context.Context, c *client.Client, applicationID int64) ([]APIJob, error) {
	var out apiJobListEnvelope
	if err := do(ctx, c, http.MethodGet, apiJobGetAllPath(applicationID), nil, &out); err != nil {
		return nil, err
	}
	return out.Schedules, nil
}

// GetAPIJob retrieves a single job by ID. There is no single-item GET
// endpoint, so this filters the full list client-side and synthesizes a 404
// APIError if not found, for consistency with client.IsNotFound.
func GetAPIJob(ctx context.Context, c *client.Client, applicationID int64, jobID string) (*APIJob, error) {
	jobs, err := ListAPIJobs(ctx, c, applicationID)
	if err != nil {
		return nil, err
	}
	for _, j := range jobs {
		if j.ID == jobID {
			return &j, nil
		}
	}
	return nil, &client.APIError{StatusCode: http.StatusNotFound, Body: "synthetic API job not found"}
}
