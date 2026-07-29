data "appdynamics_actions" "all" {
  application_id = 42
}

output "action_names" {
  value = { for a in data.appdynamics_actions.all.actions : a.id => a.name }
}
