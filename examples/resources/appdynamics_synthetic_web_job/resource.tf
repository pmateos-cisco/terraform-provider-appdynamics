resource "appdynamics_synthetic_web_job" "homepage" {
  application_id = 602
  app_key        = "AD-AAB-ABJ-HRM"
  description    = "Homepage availability check"
  url            = "https://www.example.com"

  browser_codes   = ["Chrome"]
  chrome_versions = ["86"]
  location_codes  = ["M50"]
  timeout_seconds = 30
  user_enabled    = true

  schedule_run_configs_json = jsonencode([
    {
      rate = {
        value = 15
        unit  = "MINUTES"
      }
      daysOfWeek = ["SUN", "MON", "TUES", "WED", "THUR", "FRI", "SAT"]
      timezone   = "UTC"
    }
  ])
}

# A scripted (Selenium) check instead of a simple URL load -- exactly one of
# url / script_json must be set.
resource "appdynamics_synthetic_web_job" "login_flow" {
  application_id = 602
  app_key        = "AD-AAB-ABJ-HRM"
  description    = "Login flow check"

  script_json = jsonencode({
    contentType = "INLINE_PYTHON_3"
    script      = <<-EOT
      pageUrl = "https://www.example.com/login"
      driver.get(pageUrl)
      assert "Login" in driver.title
    EOT
  })

  browser_codes   = ["Chrome"]
  chrome_versions = ["86"]
  location_codes  = ["M50"]
  timeout_seconds = 30
  user_enabled    = true

  schedule_run_configs_json = jsonencode([
    {
      rate       = { value = 1, unit = "HOURS" }
      daysOfWeek = ["SUN", "MON", "TUES", "WED", "THUR", "FRI", "SAT"]
      timezone   = "UTC"
    }
  ])
}
