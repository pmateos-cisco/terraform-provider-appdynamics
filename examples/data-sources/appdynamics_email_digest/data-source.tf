data "appdynamics_email_digest" "hourly_summary" {
  application_id  = 42
  email_digest_id = 939
}

output "hourly_summary_actions" {
  value = data.appdynamics_email_digest.hourly_summary.actions_json
}
