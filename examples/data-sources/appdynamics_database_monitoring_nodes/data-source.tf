data "appdynamics_database_monitoring_nodes" "all" {}

output "agent_versions" {
  value = { for n in data.appdynamics_database_monitoring_nodes.all.nodes : n.name => n.app_agent_version }
}
