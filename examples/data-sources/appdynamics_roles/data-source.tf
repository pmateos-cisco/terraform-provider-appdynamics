data "appdynamics_roles" "all" {}

output "role_names" {
  value = { for r in data.appdynamics_roles.all.roles : r.id => r.name }
}
