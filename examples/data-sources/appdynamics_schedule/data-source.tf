data "appdynamics_schedule" "business_hours" {
  application_id = 42
  schedule_id     = 62867
}

output "business_hours_configuration" {
  value = data.appdynamics_schedule.business_hours.schedule_configuration
}
