# A query over historical data, not a list of managed config -- results
# change over time as new violations occur.
data "appdynamics_health_rule_violations" "recent" {
  application_id   = 42
  time_range_type  = "BEFORE_NOW"
  duration_in_mins = 1440 # last 24 hours
}

output "open_violation_names" {
  value = [for v in data.appdynamics_health_rule_violations.recent.violations : v.name if v.incident_status != "RESOLVED"]
}
