resource "appdynamics_synthetic_api_application" "checkout_api" {
  name = "Checkout API Monitoring"
}

resource "appdynamics_synthetic_api_job" "health_check" {
  application_id = appdynamics_synthetic_api_application.checkout_api.id
  description    = "Checkout API health check"

  api_metadata_json = jsonencode({
    script = {
      contentType = "JAVASCRIPT"
      script      = <<-EOT
        const assert = require("assert");
        (async () => {
            var response = await client.get("https://api.example.com/health");
            assert.equal(response.statusCode, 200);
        })()
      EOT
    }
  })

  location_codes  = ["M50"]
  timeout_seconds = 15
  user_enabled    = true

  schedule_run_configs_json = jsonencode([
    {
      rate       = { value = 15, unit = "MINUTES" }
      daysOfWeek = ["SUN", "MON", "TUES", "WED", "THUR", "FRI", "SAT"]
      timezone   = "UTC"
    }
  ])
}
