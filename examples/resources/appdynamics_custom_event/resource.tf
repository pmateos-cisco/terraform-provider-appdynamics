# Create-only: any change to this config creates a brand new event rather
# than modifying the old one, and `terraform destroy` cannot delete it from
# AppDynamics (the Events API has no delete endpoint) -- it only removes the
# resource from Terraform state.
resource "appdynamics_custom_event" "deploy_marker" {
  application_id    = 42
  summary            = "Released v1.2.3"
  comment            = "Deployed via CI pipeline"
  severity           = "INFO"
  custom_event_type  = "Deployment"
  tier               = "checkout-service"

  properties = {
    version = "1.2.3"
    region  = "us-west"
  }
}
