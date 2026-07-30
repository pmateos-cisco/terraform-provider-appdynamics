# Note: unlike appdynamics_policy, email digests do not support batching
# actions -- there is no execute_actions_in_batch attribute here (the API
# rejects it as invalid for this resource type).
resource "appdynamics_email_digest" "hourly_summary" {
  application_id = 42
  name           = "Hourly Health Rule Summary"
  enabled        = true
  frequency      = 1 # hours, 1-168

  actions_json = jsonencode([
    {
      actionName = "page-oncall"
      actionType = "EMAIL"
    }
  ])

  events_json = jsonencode({
    healthRuleEvents = {
      healthRuleEventTypes = ["HEALTH_RULE_CONTINUES_CRITICAL", "HEALTH_RULE_UPGRADED"]
      healthRuleScope = {
        healthRuleScopeType = "ALL_HEALTH_RULES"
      }
    }
  })

  selected_entities_json = jsonencode({
    selectedEntityType = "ANY_ENTITY"
  })
}
