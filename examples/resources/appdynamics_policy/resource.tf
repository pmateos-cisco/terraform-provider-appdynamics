resource "appdynamics_policy" "high_cpu_alert" {
  application_id           = 42
  name                     = "High CPU Alert Policy"
  enabled                  = true
  execute_actions_in_batch = true

  actions_json = jsonencode([
    {
      actionName = appdynamics_action.page_oncall.name
      actionType = appdynamics_action.page_oncall.action_type
    }
  ])

  events_json = jsonencode({
    healthRuleEvents = {
      healthRuleEventTypes = ["POLICY_OPEN_CRITICAL", "POLICY_CONTINUES_CRITICAL"]
      healthRuleScope = {
        healthRuleScopeType = "SPECIFIC_HEALTH_RULES"
        healthRuleNames      = [appdynamics_health_rule.high_cpu.name]
      }
    }
  })

  selected_entities_json = jsonencode({
    selectedEntityType = "ANY_ENTITY"
  })
}
