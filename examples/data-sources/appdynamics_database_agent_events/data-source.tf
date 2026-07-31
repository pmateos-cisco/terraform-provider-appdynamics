data "appdynamics_database_agent_events" "recent_critical" {
  time_range_type  = "BEFORE_NOW"
  duration_in_mins = 1440
  event_types      = ["POLICY_CONTINUES_CRITICAL", "POLICY_OPEN_CRITICAL"]
  severities       = ["ERROR", "WARN"]
}

output "recent_critical_events" {
  value = data.appdynamics_database_agent_events.recent_critical.events
}
