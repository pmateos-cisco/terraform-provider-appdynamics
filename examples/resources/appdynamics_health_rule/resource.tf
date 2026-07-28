resource "appdynamics_health_rule" "high_cpu" {
  application_id = 42
  name           = "High CPU Usage"
  enabled        = true
  schedule_name  = appdynamics_schedule.business_hours.name

  affects_json = jsonencode({
    affectedEntityType = "TIER_NODE_HARDWARE"
    affectedTiers = {
      tierNames = ["Tier1"]
    }
  })

  eval_criterias_json = jsonencode({
    criticalCriteria = {
      conditionAggregationType = "ALL"
      conditions = [
        {
          name = "A"
          evalDetail = {
            evalDetailType         = "SINGLE_METRIC"
            metricPath              = "Hardware Resources|CPU|%Busy"
            metricAggregateFunction = "VALUE"
            metricEvalDetail = {
              metricEvalDetailType = "SPECIFIC_TYPE"
              compareCondition      = "GREATER_THAN_SPECIFIC_VALUE"
              compareValue           = 90
            }
          }
        }
      ]
      evalMatchingCriteria = {
        matchType = "AVERAGE"
      }
    }
  })
}
