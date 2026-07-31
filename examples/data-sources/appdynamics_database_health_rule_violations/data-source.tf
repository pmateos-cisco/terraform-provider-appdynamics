data "appdynamics_database_health_rule_violations" "primary" {
  server_id        = "512"
  time_range_type  = "BEFORE_NOW"
  duration_in_mins = 1440
}

output "violation_count" {
  value = length(data.appdynamics_database_health_rule_violations.primary.violations)
}
