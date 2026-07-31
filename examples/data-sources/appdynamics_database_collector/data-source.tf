data "appdynamics_database_collector" "orders_mysql" {
  collector_id = "616"
}

output "orders_mysql_extra_config" {
  value = data.appdynamics_database_collector.orders_mysql.extra_config_json
}
