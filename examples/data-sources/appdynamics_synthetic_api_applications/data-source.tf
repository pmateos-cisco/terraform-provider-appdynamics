data "appdynamics_synthetic_api_applications" "all" {}

output "application_ids" {
  value = { for a in data.appdynamics_synthetic_api_applications.all.applications : a.id => a.name }
}
