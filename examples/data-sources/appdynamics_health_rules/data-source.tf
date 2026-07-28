data "appdynamics_health_rules" "all" {
  application_id = 42
}

output "health_rule_names" {
  value = [for hr in data.appdynamics_health_rules.all.health_rules : hr.name]
}
