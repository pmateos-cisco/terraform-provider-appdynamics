resource "appdynamics_user" "jdoe" {
  name                    = "jdoe"
  display_name            = "Jane Doe"
  security_provider_type  = "INTERNAL"
  password                = var.jdoe_password # write-only: never read back from the API
}

variable "jdoe_password" {
  type      = string
  sensitive = true
}
