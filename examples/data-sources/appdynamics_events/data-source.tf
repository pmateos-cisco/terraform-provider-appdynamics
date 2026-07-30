# A query over historical data, not a list of managed config -- results
# change over time as new events occur.
data "appdynamics_events" "recent_deploys" {
  application_id   = 42
  time_range_type  = "BEFORE_NOW"
  duration_in_mins = 1440 # last 24 hours
  event_types      = ["APPLICATION_DEPLOYMENT", "CUSTOM"]
  severities       = ["INFO", "WARN", "ERROR"]
}

output "recent_deploy_summaries" {
  value = [for e in data.appdynamics_events.recent_deploys.events : e.summary]
}
