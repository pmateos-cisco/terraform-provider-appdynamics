package alertandrespond

import (
	"encoding/json"
	"testing"
)

func TestActionMarshalMergesExtraFields(t *testing.T) {
	a := Action{
		ID:          42,
		Name:        "page-oncall",
		ActionType:  "EMAIL",
		Notes:       "pages the on-call rotation",
		ExtraFields: json.RawMessage(`{"emails":["oncall@example.com"]}`),
	}

	b, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got["name"] != "page-oncall" {
		t.Errorf("name = %v, want page-oncall", got["name"])
	}
	if got["actionType"] != "EMAIL" {
		t.Errorf("actionType = %v, want EMAIL", got["actionType"])
	}
	if got["notes"] != "pages the on-call rotation" {
		t.Errorf("notes = %v, want pages the on-call rotation", got["notes"])
	}
	if got["id"] != float64(42) {
		t.Errorf("id = %v, want 42", got["id"])
	}
	emails, ok := got["emails"].([]any)
	if !ok || len(emails) != 1 || emails[0] != "oncall@example.com" {
		t.Errorf("emails = %v, want [oncall@example.com] merged at top level", got["emails"])
	}
}

func TestActionUnmarshalSplitsKnownAndExtraFields(t *testing.T) {
	raw := `{
		"id": 7,
		"name": "thread-dump",
		"actionType": "THREAD_DUMP",
		"numberOfThreadDumps": 5,
		"intervalInMs": 1000
	}`

	var a Action
	if err := json.Unmarshal([]byte(raw), &a); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if a.ID != 7 {
		t.Errorf("ID = %d, want 7", a.ID)
	}
	if a.Name != "thread-dump" {
		t.Errorf("Name = %q, want thread-dump", a.Name)
	}
	if a.ActionType != "THREAD_DUMP" {
		t.Errorf("ActionType = %q, want THREAD_DUMP", a.ActionType)
	}

	var extra map[string]any
	if err := json.Unmarshal(a.ExtraFields, &extra); err != nil {
		t.Fatalf("Unmarshal ExtraFields: %v", err)
	}
	if extra["numberOfThreadDumps"] != float64(5) {
		t.Errorf("numberOfThreadDumps = %v, want 5", extra["numberOfThreadDumps"])
	}
	if extra["intervalInMs"] != float64(1000) {
		t.Errorf("intervalInMs = %v, want 1000", extra["intervalInMs"])
	}
	if _, present := extra["name"]; present {
		t.Error("ExtraFields should not contain the known field 'name'")
	}
}

func TestActionRoundTrip(t *testing.T) {
	original := Action{
		Name:        "custom-jira",
		ActionType:  "CREATE_UPDATE_JIRA",
		ExtraFields: json.RawMessage(`{"jiraActionDetails":{"jiraActionType":"CREATE_JIRA"}}`),
	}

	b, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var roundTripped Action
	if err := json.Unmarshal(b, &roundTripped); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if roundTripped.Name != original.Name || roundTripped.ActionType != original.ActionType {
		t.Fatalf("round trip mismatch: got %+v, want name/type from %+v", roundTripped, original)
	}

	var extra map[string]any
	if err := json.Unmarshal(roundTripped.ExtraFields, &extra); err != nil {
		t.Fatalf("Unmarshal ExtraFields: %v", err)
	}
	details, ok := extra["jiraActionDetails"].(map[string]any)
	if !ok || details["jiraActionType"] != "CREATE_JIRA" {
		t.Errorf("jiraActionDetails = %v, want {jiraActionType: CREATE_JIRA}", extra["jiraActionDetails"])
	}
}
