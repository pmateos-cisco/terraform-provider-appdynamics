resource "appdynamics_group_membership" "jdoe_in_engineering" {
  group_id = appdynamics_group.engineering.id
  user_id  = appdynamics_user.jdoe.id
}
