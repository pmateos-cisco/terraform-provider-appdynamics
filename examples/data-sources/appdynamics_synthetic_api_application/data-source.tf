data "appdynamics_synthetic_api_application" "checkout_api" {
  application_id = "10499"
}

output "checkout_api_key" {
  value = data.appdynamics_synthetic_api_application.checkout_api.app_key
}
