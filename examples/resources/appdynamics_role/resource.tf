resource "appdynamics_role" "read_only" {
  name = "read-only"

  permissions_json = jsonencode([
    { entityType = "APPLICATION", action = "VIEW" }
  ])
}
