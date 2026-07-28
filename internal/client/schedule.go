package client

import (
	"context"
	"fmt"
	"net/http"
)

// ScheduleConfiguration is a union of the fields used by each scheduleFrequency
// value; only the fields relevant to ScheduleFrequency should be set.
type ScheduleConfiguration struct {
	ScheduleFrequency string   `json:"scheduleFrequency"`
	StartDate         string   `json:"startDate,omitempty"`
	StartTime         string   `json:"startTime,omitempty"`
	EndDate           string   `json:"endDate,omitempty"`
	EndTime           string   `json:"endTime,omitempty"`
	Days              []string `json:"days,omitempty"`
	Day               []string `json:"day,omitempty"`
	Occurrence        string   `json:"occurrence,omitempty"`
	StartCron         string   `json:"startCron,omitempty"`
	EndCron           string   `json:"endCron,omitempty"`
}

type Schedule struct {
	ID                    int64                  `json:"id,omitempty"`
	Name                  string                 `json:"name"`
	Description           string                 `json:"description,omitempty"`
	Timezone              string                 `json:"timezone"`
	ScheduleConfiguration *ScheduleConfiguration `json:"scheduleConfiguration"`
}

func schedulesPath(applicationID int64) string {
	return fmt.Sprintf("/controller/alerting/rest/v1/applications/%d/schedules", applicationID)
}

func schedulePath(applicationID, scheduleID int64) string {
	return fmt.Sprintf("%s/%d", schedulesPath(applicationID), scheduleID)
}

func (c *Client) CreateSchedule(ctx context.Context, applicationID int64, s *Schedule) (*Schedule, error) {
	var out Schedule
	if err := c.do(ctx, http.MethodPost, schedulesPath(applicationID), s, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetSchedule(ctx context.Context, applicationID, scheduleID int64) (*Schedule, error) {
	var out Schedule
	if err := c.do(ctx, http.MethodGet, schedulePath(applicationID, scheduleID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateSchedule(ctx context.Context, applicationID, scheduleID int64, s *Schedule) (*Schedule, error) {
	var out Schedule
	if err := c.do(ctx, http.MethodPut, schedulePath(applicationID, scheduleID), s, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteSchedule(ctx context.Context, applicationID, scheduleID int64) error {
	return c.do(ctx, http.MethodDelete, schedulePath(applicationID, scheduleID), nil, nil)
}
