resource "appdynamics_role_group_assignment" "engineering_read_only" {
  role_id  = appdynamics_role.read_only.id
  group_id = appdynamics_group.engineering.id
}
