data "appdynamics_schedules" "all" {
  application_id = 42
}

output "schedule_names" {
  value = { for s in data.appdynamics_schedules.all.schedules : s.id => s.name }
}
