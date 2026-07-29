# Look up by ID:
data "appdynamics_action_suppression" "by_id" {
  application_id        = 42
  action_suppression_id = 150
}

# ...or by name (exactly one of action_suppression_id / name must be set):
data "appdynamics_action_suppression" "by_name" {
  application_id = 42
  name           = "Weekend Maintenance"
}

output "maintenance_window_affects" {
  value = data.appdynamics_action_suppression.by_name.affects_json
}
