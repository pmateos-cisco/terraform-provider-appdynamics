data "appdynamics_synthetic_web_jobs" "all" {
  application_id = 602
}

output "job_ids" {
  value = { for j in data.appdynamics_synthetic_web_jobs.all.jobs : j.id => j.description }
}
