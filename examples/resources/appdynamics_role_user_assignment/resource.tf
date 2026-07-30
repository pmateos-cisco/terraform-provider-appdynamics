resource "appdynamics_role_user_assignment" "jdoe_read_only" {
  role_id = appdynamics_role.read_only.id
  user_id = appdynamics_user.jdoe.id
}
