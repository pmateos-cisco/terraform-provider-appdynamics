# Look up by ID:
data "appdynamics_user" "by_id" {
  user_id = 3567
}

# ...or by name (exactly one of user_id / name must be set):
data "appdynamics_user" "by_name" {
  name = "jdoe"
}

output "jdoe_display_name" {
  value = data.appdynamics_user.by_name.display_name
}
