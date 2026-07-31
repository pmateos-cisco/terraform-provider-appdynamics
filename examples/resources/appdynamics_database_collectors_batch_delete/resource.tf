# Permanently deletes these three collectors in one call, as soon as this
# resource is created. There is no way to un-delete them: terraform destroy
# only forgets this resource, it does not restore the collectors.
resource "appdynamics_database_collectors_batch_delete" "decommission" {
  collector_ids = [101, 102, 103]
}
