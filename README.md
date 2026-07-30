# terraform-provider-appdynamics

A Terraform provider for Splunk AppDynamics, covering the "Alert and Respond" Platform APIs.

Targets AppDynamics SaaS controllers via OAuth2 client-credentials auth.

## Project structure

Go source is grouped by AppDynamics API family under a subpackage named for that family:

- `internal/client/alertandrespond/` — REST client for the [Alert and Respond
  API](https://help.splunk.com/en/appdynamics-on-premises/extend-appdynamics/26.7.0/extend-splunk-appdynamics/splunk-appdynamics-apis/alert-and-respond-api)
  (schedules, actions, health rules, policies, action suppressions).
- `internal/provider/alertandrespond/` — the Terraform resources/data sources backed by that
  client.
- `internal/provider/provider.go` stays at the top level — it's the single `provider.Provider`
  implementation, and just registers every API family's resources/data sources into one provider.

As more AppDynamics API families get added, each gets its own sibling subpackage (e.g.
`internal/client/<family>/`, `internal/provider/<family>/`) rather than growing the existing ones.

Terraform's own conventions — `examples/resources/<type>/`, `examples/data-sources/<type>/`,
`docs/resources/*.md`, `docs/data-sources/*.md` — stay flat regardless of API family, since
`tfplugindocs` and the Registry expect that exact layout.

## Resources

- `appdynamics_schedule` — alerting schedules
- `appdynamics_action` — alert actions (email, thread dump, HTTP request, JIRA, etc.)
- `appdynamics_health_rule` — health rules (create/read/update/delete a single rule)
- `appdynamics_health_rules_enable_all` — bulk-enables every health rule for an application on
  create, bulk-disables them on destroy. Models a one-shot action rather than a tracked object:
  `Read`/`Update` are no-ops, and `application_id` forces replacement on change, so re-running the
  enable call requires tainting/replacing the resource rather than a routine `apply`. See
  `examples/resources/appdynamics_health_rules_enable_all`.
- `appdynamics_policy` — policies binding events to actions
- `appdynamics_action_suppression` — schedules during which actions are muted for a scope of
  entities (a maintenance window). `suppression_schedule_type` is `ONE_TIME` (uses `start_time`/
  `end_time`) or `RECURRING` (uses `recurring_schedule_json`, same `scheduleFrequency` shapes as
  `appdynamics_schedule`'s `schedule_configuration`).
- `appdynamics_deployment_event` / `appdynamics_custom_event` — create an `APPLICATION_DEPLOYMENT`
  or `CUSTOM` event via the (legacy, non-`/alerting/rest/v1/`) Events API. **Create-only**: events
  are immutable log entries with no "get by ID", update, or delete support in the underlying API,
  so every attribute forces replacement on change, `Read` never contacts the API (nothing reliable
  to read back), and `terraform destroy` only removes the resource from Terraform state — with a
  warning — since the event itself can't be deleted from AppDynamics.
- `appdynamics_email_digest` — a periodic rollup email binding trigger events on a set of entities
  to a set of actions, same shape as `appdynamics_policy` (`actions_json`/`events_json`/
  `selected_entities_json`) plus `frequency` (hours, 1-168). Unlike `appdynamics_policy`, there is
  **no `execute_actions_in_batch`** — the API rejects that field as invalid for email digests
  (verified live), even though the official docs' own example payload includes it.

## Data Sources

- `appdynamics_health_rules` — lists health rules for an application (`id`, `name`, `enabled`,
  `affected_entity_type` only; use the singular data source below for full detail on one rule).
- `appdynamics_health_rule` — retrieves the full detail (including `affects_json` /
  `eval_criterias_json`) of one health rule by `application_id` + `health_rule_id`. Shares its type
  name with the managed resource above — `resource "appdynamics_health_rule"` and
  `data "appdynamics_health_rule"` are separate namespaces, so this is expected, not a conflict.
- `appdynamics_actions` — lists actions for an application (`id`, `name`, `action_type` only).
- `appdynamics_action_suppressions` — lists action suppressions for an application (`id`, `name`,
  `timezone`, `start_time`, `end_time` only).
- `appdynamics_action_suppression` — retrieves full detail of one action suppression, looked up by
  **either** `action_suppression_id` or `name` (exactly one must be set) — the only data source in
  this provider backed by two different lookup endpoints.
- `appdynamics_policies` — lists policies for an application (`id`, `name`, `enabled` only; the
  list endpoint actually returns richer detail per policy, but only these three fields are surfaced
  for consistency with this provider's other list data sources).
- `appdynamics_policy` — retrieves the full detail (including `actions_json` / `events_json` /
  `selected_entities_json`) of one policy by `application_id` + `policy_id`. Shares its type name
  with the managed resource, same as `appdynamics_health_rule`.
- `appdynamics_events` / `appdynamics_health_rule_violations` — query the (legacy) Events API for
  events or health rule violations within a time range. Unlike this provider's other data sources,
  these are **reporting/lookup queries over historical data, not lists of current config** — the
  same query can return different results each time it's run as new events/violations occur.
- `appdynamics_schedules` — lists schedules for an application (`id`, `name`, `description`,
  `timezone` only — no `schedule_configuration`; use the singular data source below for that).
- `appdynamics_schedule` — retrieves the full detail (including `schedule_configuration`) of one
  schedule by `application_id` + `schedule_id`. Shares its type name with the managed resource,
  same as `appdynamics_health_rule`.
- `appdynamics_email_digests` — lists email digests for an application (`id`, `name`, `enabled`
  only).
- `appdynamics_email_digest` — retrieves the full detail (including `actions_json` / `events_json`
  / `selected_entities_json` / `frequency`) of one email digest by `application_id` +
  `email_digest_id`. Shares its type name with the managed resource, same as
  `appdynamics_health_rule`.

## Known API documentation gaps

### Action item paths

The Actions API docs describe `GET`/single-item retrieval at `/actions/{action-id}` (plural) but
`PUT`/`DELETE` at `/action/{action-id}` (singular). Verified live against a real controller: **all
three (GET, PUT, DELETE) actually require the plural `/actions/{action-id}` form** — the singular
form 404s across the board. `internal/client/action.go`'s `actionItemPath` uses the plural form for
all three operations, contradicting the docs' claim for PUT/DELETE but matching observed behavior.

### Action suppression field names

The Action Suppression API docs mention `healthRuleScope.healthRuleScopeType` but never give the
field name for the actual list of health rule names. Verified live: the real field is
**`healthRules`**, not `healthRuleNames`. A `RECURRING` suppression's `recurringSchedule` block
also isn't documented at all beyond the type name — verified live that it reuses the same
`scheduleFrequency`-discriminated shape as `appdynamics_schedule`'s `schedule_configuration`
(`scheduleFrequency: "WEEKLY"`, `days`, `startTime`, `endTime`, etc.).

### Events API request/response format and indexing delay

The Events API docs show an XML response and don't mention that `POST` takes URL-encoded form
parameters (not JSON) or that its response is plain text (`"Successfully created the event
id:1234"`) even when `output=JSON` is requested — that param only affects `GET` queries. Multiple
`propertynames`/`propertyvalues` must be sent as **repeated** form parameters
(`propertynames=a&propertynames=b`), not comma-joined (`propertynames=a,b` is treated as one
literal property name containing a comma).

Also verified live: there's a real indexing delay between creating an event and it becoming
visible through the query endpoint — in testing, a freshly created event was briefly invisible to
`appdynamics_events` (empty result) before showing up correctly on a later query. This isn't a bug
in this provider; treat `appdynamics_events`/`appdynamics_health_rule_violations` results as
eventually consistent, not immediate.

### Email Digest API GET-list path and executeActionsInBatch

The docs show the list endpoint at `/controller/alerting/rest/v1/applications/<application_id>/`
(no `email-digests` segment) — that's a documentation typo; the real path is
`/applications/<application_id>/email-digests`, verified live. Separately, the docs' own example
create payload includes `"executeActionsInBatch": true`, but the API rejects it outright
(`400: Execute actions in batch is not allowed for email digests`) — unlike `appdynamics_policy`,
`appdynamics_email_digest` has no `execute_actions_in_batch` attribute at all.

## Design note: JSON passthrough for nested config

The AppDynamics APIs have deeply polymorphic nested blocks — a health rule's `evalCriterias`
branches by metric-vs-expression and baseline-vs-specific-value; a policy's `events` and
`selectedEntities` vary by event/entity type; an action's extra fields vary by `actionType`.
Rather than modeling every union as Terraform nested attributes (a much larger schema for the
same functionality), those blocks are exposed as `jsonencode()`'d JSON string attributes
(`affects_json`, `eval_criterias_json`, `events_json`, `selected_entities_json`, `extra_fields`,
`actions_json`) that are passed straight through to the Controller API. See `examples/` for the
JSON shapes, and the [Splunk AppDynamics Platform API docs](https://help.splunk.com/en/appdynamics-on-premises/extend-appdynamics/25.7.0/extend-splunk-appdynamics/splunk-appdynamics-apis/platform-api-index)
for the full reference. These attributes use `jsontypes.Normalized`, so formatting differences
(key order, whitespace) don't cause spurious diffs.

Note that the Controller API echoes these blocks back with extra server-defaulted fields that were
never in the request (e.g. `evaluateToTrueOnNoData`, `violationStatusOnNoData`, `warningCriteria`,
`suppressionMaintenanceType`). Since `affects_json`/`eval_criterias_json`/`events_json`/
`selected_entities_json`/`recurring_schedule_json`/`health_rule_scope_json` are `Required` or
`Optional` (not `Computed`), `Create`/`Update`/`Read` keep the plan's or prior state's own value for
these attributes rather than the API's expanded response — otherwise Terraform either flags an
"inconsistent result after apply" error or shows a perpetual spurious diff on every `plan`.

## API Response Codes

Response codes returned by the underlying AppDynamics Controller REST API (surfaced to Terraform
as the `status <code>` in `appdynamics api error: status <code>: <body>`):

| Code | Description |
| ---- | ----------- |
| 200 | Fetched successfully |
| 201 | Created successfully |
| 204 | Deleted successfully |
| 400 | Bad request |
| 401 | Unauthorized |
| 403 | Forbidden |
| 404 | Resource not found |
| 409 | Already exists |

## Building

Requires Go 1.23+.

```sh
make build      # builds ./terraform-provider-appdynamics.exe
make vet fmt    # go vet + gofmt -s
make test       # unit tests (internal/client, no network)
make testacc    # acceptance tests against a real controller (TF_ACC=1; not yet written)
```

`make` isn't installed by default on Windows — if you don't have it, run the underlying Go
commands directly, e.g. `go build -o terraform-provider-appdynamics.exe .` and `go test ./...`.

## Trying it against a real controller (local dev override)

1. Build and install the binary onto your `GOPATH/bin`:

   ```sh
   make install
   ```

2. Create (or edit) `%APPDATA%\terraform.rc` (Windows) / `~/.terraformrc` (macOS/Linux) with a
   `dev_overrides` block pointing at your `GOPATH/bin`:

   ```hcl
   provider_installation {
     dev_overrides {
       "pmateos-cisco/appdynamics" = "C:\\Users\\<you>\\go\\bin"
     }
     direct {}
   }
   ```

   Run `go env GOPATH` to confirm the exact path.

3. In a test `.tf` file, set provider credentials (from an AppDynamics API Client with the
   permissions needed for the resources you're managing):

   ```sh
   export APPD_CONTROLLER_URL="https://mycompany.saas.appdynamics.com"
   export APPD_CLIENT_ID="my-api-client@mycompany"
   export APPD_CLIENT_SECRET="..."
   ```

   Use `examples/resources/*/resource.tf` as a starting point — a schedule, an action, a health
   rule that references the schedule, and a policy that references the action and health rule.

4. With `dev_overrides` active, Terraform uses your locally built binary directly — skip
   `terraform init` and run `terraform plan` / `terraform apply` straight away.

## Generating docs

```sh
make docs
```

Reads `examples/` and each resource's `Description`/attribute descriptions to populate `docs/`.
