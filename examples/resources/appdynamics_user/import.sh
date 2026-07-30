# Import ID is just the user's numeric ID -- RBAC entities are account-wide,
# not scoped to an application_id like every other resource in this provider.
terraform import appdynamics_user.jdoe 3567
