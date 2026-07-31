data "appdynamics_database_collectors" "all" {}

output "collector_ids" {
  value = { for c in data.appdynamics_database_collectors.all.collectors : c.id => c.name }
}
