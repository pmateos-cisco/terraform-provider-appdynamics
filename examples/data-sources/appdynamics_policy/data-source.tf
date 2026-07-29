data "appdynamics_policy" "example" {
  application_id = 42
  policy_id      = 937
}

output "policy_actions" {
  value = data.appdynamics_policy.example.actions_json
}
