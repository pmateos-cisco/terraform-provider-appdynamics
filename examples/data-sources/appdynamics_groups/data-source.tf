data "appdynamics_groups" "all" {}

output "group_names" {
  value = { for g in data.appdynamics_groups.all.groups : g.id => g.name }
}
