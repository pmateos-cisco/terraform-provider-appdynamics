data "appdynamics_synthetic_api_job" "health_check" {
  application_id = 10499
  job_id         = "4af8bead-8ad9-4668-a073-4accf2f48079"
}

output "health_check_schedule" {
  value = data.appdynamics_synthetic_api_job.health_check.schedule_run_configs_json
}
