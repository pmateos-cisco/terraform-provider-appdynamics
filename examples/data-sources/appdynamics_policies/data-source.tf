data "appdynamics_policies" "all" {
  application_id = 42
}

output "policy_names" {
  value = { for p in data.appdynamics_policies.all.policies : p.id => p.name }
}
