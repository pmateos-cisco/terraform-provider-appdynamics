resource "appdynamics_health_rule" "high_cpu" {
  application_id                = 42
  name                          = "High CPU Usage"
  enabled                       = true
  schedule_name                 = appdynamics_schedule.business_hours.name
  use_data_from_last_n_minutes  = 15
  wait_time_after_violation     = 5

  affects_json = jsonencode({
    affectedEntityType = "TIER_NODE_HARDWARE"
    affectedEntities = {
      tierOrNode = "TIER_AFFECTED_ENTITIES"
      affectedTiers = {
        affectedTierScope = "SPECIFIC_TIERS"
        tiers             = ["Tier1"]
        shouldNot         = false
      }
    }
  })

  eval_criterias_json = jsonencode({
    criticalCriteria = {
      conditionAggregationType = "ALL"
      conditions = [
        {
          name      = "A"
          shortName = "A"
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
