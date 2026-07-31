data "appdynamics_synthetic_web_job" "homepage" {
  application_id = 602
  job_id         = "4af8bead-8ad9-4668-a073-4accf2f48079"
}

output "homepage_schedule" {
  value = data.appdynamics_synthetic_web_job.homepage.schedule_run_configs_json
}
