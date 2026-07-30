data "appdynamics_users" "all" {}

output "user_names" {
  value = { for u in data.appdynamics_users.all.users : u.id => u.name }
}
