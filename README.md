# terraform-provider-appdynamics

A Terraform provider for Splunk AppDynamics, covering the "Alert and Respond" Platform APIs, the
RBAC (role-based access control) API, and Synthetic Web Monitoring jobs.

Targets AppDynamics SaaS controllers via OAuth2 client-credentials auth.

## Project structure

Go source is grouped by AppDynamics API family under a subpackage named for that family:

- `internal/client/alertandrespond/` — REST client for the [Alert and Respond
  API](https://help.splunk.com/en/appdynamics-on-premises/extend-appdynamics/26.7.0/extend-splunk-appdynamics/splunk-appdynamics-apis/alert-and-respond-api)
  (schedules, actions, health rules, policies, action suppressions).
- `internal/provider/alertandrespond/` — the Terraform resources/data sources backed by that
  client.
- `internal/client/rbac/` / `internal/provider/rbac/` — sibling package for the RBAC
  (role-based access control) API family: users, groups, roles, and the associations between them.
  Reuses the `alertandrespond` package's OAuth-token-caching HTTP client via its exported `DoTyped`
  method (RBAC requires a different, versioned `Content-Type`/`Accept` header) rather than
  duplicating the client.
- `internal/client/synthetics/` / `internal/provider/synthetics/` — sibling package for Synthetic
  Web Monitoring jobs. Also reuses the `alertandrespond` client's `DoTyped` (standard
  `application/json`, no special content type needed), but talks to a different base path: the
  Controller's own internal `/controller/restui/synthetic/...` UI-backing API rather than the
  officially documented Synthetic Monitoring API, which requires a separate EUM account
  username/license key credential pair this provider doesn't use. See "Known API documentation
  gaps" below.
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
- `appdynamics_user` — an RBAC user account. `name`/`display_name` are mutable in place; `password`
  is write-only and forces replacement on change (the API has no password-update endpoint at all —
  verified live, see below).
- `appdynamics_group` — an RBAC group. `name`/`description` are mutable in place.
- `appdynamics_role` — an RBAC role (name + `permissions_json`). `name` is mutable in place;
  `permissions_json` forces replacement on change, since the API cannot update permissions on an
  existing role (verified live, see below).
- `appdynamics_group_membership` (`group_id`, `user_id`) / `appdynamics_role_user_assignment`
  (`role_id`, `user_id`) / `appdynamics_role_group_assignment` (`role_id`, `group_id`) — dedicated
  association resources modeling the many-to-many membership/assignment toggles, rather than list
  attributes on the parent resource. Both ID fields force replacement on change; there's no partial
  update for an association, only create/delete.
- `appdynamics_synthetic_web_job` — a Synthetic Web Monitoring job (a scheduled browser check,
  either a simple `url` page-load or a Selenium `script_json`; exactly one of the two must be set).
  `application_id` and `app_key` (the EUM Browser App key) force replacement on change; every other
  attribute, including `schedule_run_configs_json`, updates in place. Unlike
  `appdynamics_custom_event`/`appdynamics_deployment_event`, this **is** a real, full CRUD resource
  — a working (undocumented) delete endpoint was found and verified live, see below.
- `appdynamics_synthetic_api_application` — a Synthetic API Monitoring application: the lightweight,
  account-wide container `appdynamics_synthetic_api_job` resources are grouped under. Unlike Web
  Monitoring's Browser RUM app requirement, this container has a full create/delete API lifecycle
  (verified live). `name` forces replacement on change (no update endpoint was found).
- `appdynamics_synthetic_api_job` — a Synthetic API Monitoring job (a scheduled scripted API check,
  `api_metadata_json`). `application_id` forces replacement on change; every other attribute,
  including `schedule_run_configs_json`, updates in place. No `app_key` attribute — verified live
  that omitting it lets the server auto-assign the key tied to `application_id`.

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
- `appdynamics_users` / `appdynamics_groups` / `appdynamics_roles` — list all users/groups/roles in
  the account (RBAC entities are account-wide, not scoped to an `application_id`, unlike every
  Alert-and-Respond data source above).
- `appdynamics_user` / `appdynamics_group` — retrieve full detail of one user/group, looked up by
  **either** its numeric ID or `name` (exactly one must be set), same dual-lookup pattern as
  `appdynamics_action_suppression`.
- `appdynamics_role` — retrieves full detail (including `permissions_json`) of one role by
  `role_id` only (no by-name lookup, since permissions require the ID-based
  `include-permissions=true` GET).
- `appdynamics_synthetic_web_jobs` — lists Synthetic Web Monitoring jobs for a business application
  (`id`, `description`, `url`, `user_enabled` only).
- `appdynamics_synthetic_web_job` — retrieves the full detail (including
  `schedule_run_configs_json` / `script_json` / `network_profile_json` / etc.) of one job by
  `application_id` + `job_id`. Shares its type name with the managed resource, same as
  `appdynamics_health_rule`.
- `appdynamics_synthetic_api_applications` — lists all Synthetic API Monitoring applications in the
  account (`id`, `name` only; account-wide, no inputs).
- `appdynamics_synthetic_api_application` — retrieves the detail (including `app_key`) of one
  application by ID. Shares its type name with the managed resource.
- `appdynamics_synthetic_api_jobs` — lists Synthetic API Monitoring jobs for an application (`id`,
  `description`, `user_enabled` only).
- `appdynamics_synthetic_api_job` — retrieves the full detail (including
  `schedule_run_configs_json` / `api_metadata_json` / etc.) of one job by `application_id` +
  `job_id`. Shares its type name with the managed resource.

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

### RBAC API content type, not-found behavior, and immutable fields

The RBAC API docs don't mention that both the `Content-Type` **and** `Accept` headers must be set
to the versioned `application/vnd.appd.cntrl+json;v=1` — a plain `application/json` request gets
`415 Unsupported Media Type`, and setting the versioned type on `Content-Type` alone (leaving
`Accept` as JSON) gets `406 Not Acceptable`. Verified live.

Also verified live: `GET` on a deleted/nonexistent user, group, or role returns a raw HTTP `500`
with a Java NPE-style message (e.g. `Cannot invoke "...User.getId()" because "user" is null"`), not
a `404`. `internal/client/rbac.IsNotFound` treats both the standard 404 and this 500-with-`"is
null"` pattern as "not found" so `Read`/`Delete` behave correctly.

Two fields are silently rejected on update, contradicting/omitting this from the docs:

- `PUT /users/{id}` returns `400: 'password' should not specify` if the request body includes a
  `password` field **at all**, even resending the same value unchanged — there is no
  password-update endpoint in this API. `appdynamics_user`'s `password` attribute is write-only and
  forces replacement on change.
- `PUT /roles/{id}` returns `500: Users are not allowed to create/update permissions` if the
  request body includes a `permissions` field at all. The docs do state elsewhere that role update
  is limited to `name`/`description`, but don't mention that merely *including* the field (even
  unchanged) triggers a hard error rather than being ignored. `appdynamics_role`'s
  `permissions_json` attribute forces replacement on change.

Also verified live: a user's `name` (unlike role permissions or password) genuinely is mutable via
`PUT`, and both `id` and `security_provider_type` must be present in every `PUT /users/{id}` body
even though the docs only mention `name`/`displayName` as updatable — omitting either gets
`"id is not match"` or `"'security_provider_type' need to be internal"`.

Group membership is reflected only on the **user's** detail response (`groups: [...]`), never on
the group; role assignment is reflected on **both** the user's and the group's detail response
(`roles: [...]`) depending on whether it's a user or group assignment. The three association
resources' `Read` each query whichever entity actually reflects that specific relationship.

### Synthetic Monitoring API: no usable public endpoint, EUM app key discovery, and an undocumented delete

The officially documented Synthetic Monitoring API (`<api_server_URL>/v1/synthetic/schedule`,
Basic auth with an EUM account username + license key) is a separate credential pair this provider
doesn't collect, so `appdynamics_synthetic_web_job` instead uses the Controller's own internal
`/controller/restui/synthetic/schedule/<applicationId>/...` API — the same one the Controller UI
itself calls — via OAuth with the existing client credentials. Verified live, but comes with its
own set of undocumented behavior:

- `application_id` in the URL path is **not** the Browser RUM app's own ID — there's no API to look
  it up. It has to come from the Controller UI (User Experience > Browser Apps > *App Name*, visible
  in the URL). Using the wrong ID, or an APM application ID, either 500s with a Java NPE (if that ID
  has no EUM config at all) or fails create with `"Web Monitoring job can't be created under this
  application"` (if it has EUM config but isn't the right Browser App).
- `app_key` (the EUM Browser App key, e.g. `AD-AAB-ABJ-HRM`, also only visible in the Controller UI)
  must match the Browser App tied to that `application_id`.
- Create/update both return **`204 No Content` with no body** — contradicting the docs' documented
  200-with-full-object response. Since there's also no single-item GET (`/getSchedule/{id}` 404s,
  verified live), `appdynamics_synthetic_web_job`'s `Create` re-lists all jobs and matches on
  `description`+`appKey` to find the new job's ID; `Read` always goes through the list endpoint and
  filters client-side.
- A newly created job defaults to **`userEnabled: false`** (disabled) unless explicitly set `true`
  in the request — the docs' example response shows `userEnabled: true` with no mention that this
  isn't the actual default.
- `rate.value` must be 1-59 for a `MINUTES`-unit schedule (e.g. `60` is rejected with
  `"rate.value must be between 1 & 59 for MINUTES"` — use `{value: 1, unit: "HOURS"}` instead).
- A working delete endpoint exists — `POST .../deleteSchedules` with a JSON array of job IDs in the
  body, returning `204` — despite not being listed anywhere in the docs' index of Synthetic
  Monitoring API endpoints (which only lists create/update/get, no delete). This was found by
  inspecting the Controller UI's own network calls, not by anything in the documentation.
- Synthetic API Monitoring jobs (`appdynamics_synthetic_api_job`) are **not** grouped under a
  Browser RUM app the way Web Monitoring jobs are, despite using the same `updateSchedule`/
  `getSchedules`/`deleteSchedules` endpoint family under `.../synthetic/api-schedule/{applicationId}/
  ...` instead of `.../synthetic/schedule/{applicationId}/...`. The initial attempt to create one
  under a Browser RUM app's ID failed with `"API Monitoring job can't be created under this
  application"` regardless of payload — the real requirement is a separate, dedicated container
  entity of type `SYNTH_API_MONITORING`, created via `POST
  /controller/restui/allApplications/createApplication?applicationType=SYNTH_API_MONITORING` with
  body `{"name": "..."}` (found by inspecting the Controller UI's own network calls, not documented
  anywhere). This container is modeled as `appdynamics_synthetic_api_application`:
  - `createApplication`'s response doesn't include `appKey` (`applicationTypeInfo`/`eumAppName` are
    null immediately after create) -- the full record, including `appKey`, is only available via
    `GET /controller/restui/eumApplications/getEumApiMonitoringApplications`, an account-wide list
    with no per-ID GET (verified live: several plausible single-item paths all 404).
  - Deletion is `POST /controller/restui/allApplications/deleteApplication` with a **bare JSON
    number** as the body (`10500`, not `[10500]` or `{"id":10500}` -- both rejected with a
    deserialization error naming the expected type as `long`), returning `204`.
  - No update endpoint was found (`POST .../updateApplication` 404s), so
    `appdynamics_synthetic_api_application`'s `name` forces replacement on change.
  - Once the application exists, `appdynamics_synthetic_api_job` creation needs **no `appKey` at
    all** in the request body — verified live that omitting it lets the server auto-assign the key
    tied to the target `applicationId`; the earlier failure was caused by pointing at the wrong kind
    of application, not a missing/wrong key.
  - **Caution**: during testing, calling `deleteSchedules` for the last remaining job under a
    long-lived, pre-existing `SYNTH_API_MONITORING` application was followed by that application's
    `getSchedules` endpoint permanently returning `500 InternalServerException` on both the
    api-schedule and web-schedule paths, where it had been working normally moments before. The
    application was separately removed via the Controller UI before root cause could be isolated, so
    it's unconfirmed whether the delete call actually caused this or it was a coincidental
    environment issue -- but a repeat of the same sequence (create app → create job → delete job)
    against a disposable test application afterward completed cleanly with no such error. Treat
    deleting the last job under a long-lived application with some caution until this is better
    understood.

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
