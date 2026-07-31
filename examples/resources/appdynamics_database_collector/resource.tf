resource "appdynamics_database_collector" "orders_mysql" {
  type       = "MYSQL"
  name       = "orders-mysql"
  hostname   = "orders-db.internal.example.com"
  port       = 3306
  username   = "appd_monitor"
  password   = var.orders_mysql_password
  agent_name = "orders-db-agent"
  enabled    = true

  extra_config_json = jsonencode({
    databaseName    = "orders"
    loggingEnabled  = false
    removeLiterals  = true
  })
}

variable "orders_mysql_password" {
  type      = string
  sensitive = true
}
