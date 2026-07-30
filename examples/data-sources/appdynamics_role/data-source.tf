data "appdynamics_role" "read_only" {
  role_id = 2695
}

output "read_only_permissions" {
  value = data.appdynamics_role.read_only.permissions_json
}
