resource "appdynamics_schedule" "business_hours" {
  application_id = 42
  name           = "Business Hours"
  description    = "Weekdays 9am-6pm Pacific"
  timezone       = "America/Los_Angeles"

  schedule_configuration = {
    schedule_frequency = "WEEKLY"
    days                = ["MONDAY", "TUESDAY", "WEDNESDAY", "THURSDAY", "FRIDAY"]
    start_time          = "09:00"
    end_time            = "18:00"
  }
}
