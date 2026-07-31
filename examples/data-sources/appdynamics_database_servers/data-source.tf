data "appdynamics_database_servers" "all" {}

output "server_names" {
  value = { for s in data.appdynamics_database_servers.all.servers : s.id => s.name }
}
