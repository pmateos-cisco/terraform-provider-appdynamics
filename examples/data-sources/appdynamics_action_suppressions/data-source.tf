data "appdynamics_action_suppressions" "all" {
  application_id = 42
}

output "action_suppression_names" {
  value = { for as in data.appdynamics_action_suppressions.all.action_suppressions : as.id => as.name }
}
