# terraform-provider-appdynamics

A Terraform provider for Splunk AppDynamics, covering the "Alert and Respond" Platform APIs.

Targets AppDynamics SaaS controllers via OAuth2 client-credentials auth.

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

## Data Sources

- `appdynamics_health_rules` — lists health rules for an application (`id`, `name`, `enabled`,
  `affected_entity_type` only; use the singular data source below for full detail on one rule).
- `appdynamics_health_rule` — retrieves the full detail (including `affects_json` /
  `eval_criterias_json`) of one health rule by `application_id` + `health_rule_id`. Shares its type
  name with the managed resource above — `resource "appdynamics_health_rule"` and
  `data "appdynamics_health_rule"` are separate namespaces, so this is expected, not a conflict.

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
never in the request (e.g. `evaluateToTrueOnNoData`, `violationStatusOnNoData`, `warningCriteria`).
Since `affects_json`/`eval_criterias_json`/`events_json`/`selected_entities_json` are `Required`
(not `Computed`), `Create`/`Update`/`Read` keep the plan's or prior state's own value for these
attributes rather than the API's expanded response — otherwise Terraform either flags an
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
       "pmateos/appdynamics" = "C:\\Users\\<you>\\go\\bin"
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
