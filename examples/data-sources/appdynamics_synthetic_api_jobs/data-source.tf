data "appdynamics_synthetic_api_jobs" "all" {
  application_id = 10499
}

output "job_ids" {
  value = { for j in data.appdynamics_synthetic_api_jobs.all.jobs : j.id => j.description }
}
