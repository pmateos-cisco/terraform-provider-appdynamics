resource "appdynamics_action_suppression" "maintenance_window" {
  application_id             = 42
  name                       = "Weekend Maintenance"
  disable_agent_reporting    = false
  suppression_schedule_type  = "RECURRING"
  timezone                   = "America/Los_Angeles"

  affects_json = jsonencode({
    affectedInfoType = "APPLICATION"
  })

  # Same scheduleFrequency shapes as appdynamics_schedule's schedule_configuration.
  recurring_schedule_json = jsonencode({
    scheduleFrequency = "WEEKLY"
    days              = ["SATURDAY", "SUNDAY"]
    startTime         = "09:00"
    endTime           = "10:00"
  })
}

resource "appdynamics_action_suppression" "one_time_window" {
  application_id             = 42
  name                       = "Release Deploy Window"
  suppression_schedule_type  = "ONE_TIME"
  timezone                   = "America/Los_Angeles"
  start_time                 = "2026-08-01T11:43:00"
  end_time                   = "2026-08-01T12:43:00"

  affects_json = jsonencode({
    affectedInfoType = "APPLICATION"
  })

  health_rule_scope_json = jsonencode({
    healthRuleScopeType = "SPECIFIC_HEALTH_RULES"
    healthRules         = ["High CPU Usage"]
  })
}
