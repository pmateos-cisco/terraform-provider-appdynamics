data "appdynamics_health_rule" "high_cpu" {
  application_id = 42
  health_rule_id = 205
}

output "high_cpu_affects" {
  value = data.appdynamics_health_rule.high_cpu.affects_json
}
