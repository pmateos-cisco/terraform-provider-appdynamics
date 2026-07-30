# Look up by ID:
data "appdynamics_group" "by_id" {
  group_id = 77
}

# ...or by name (exactly one of group_id / name must be set):
data "appdynamics_group" "by_name" {
  name = "engineering"
}

output "engineering_description" {
  value = data.appdynamics_group.by_name.description
}
