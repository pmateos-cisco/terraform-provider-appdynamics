resource "appdynamics_action" "page_oncall" {
  application_id = 42
  name           = "page-oncall"
  action_type    = "EMAIL"

  extra_fields = jsonencode({
    emails = ["oncall@example.com"]
  })
}

resource "appdynamics_action" "capture_thread_dumps" {
  application_id = 42
  name           = "capture-thread-dumps"
  action_type    = "THREAD_DUMP"

  extra_fields = jsonencode({
    numberOfThreadDumps = 5
    intervalInMs         = 1000
    approvalBeforeExecution = {
      requireApproval = false
    }
  })
}
