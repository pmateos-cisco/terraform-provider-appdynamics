resource "appdynamics_group" "engineering" {
  name                   = "engineering"
  description            = "Engineering team"
  security_provider_type = "INTERNAL"
}
