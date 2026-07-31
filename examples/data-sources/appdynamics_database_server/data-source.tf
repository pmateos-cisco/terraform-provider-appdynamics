data "appdynamics_database_server" "primary" {
  server_id = "512"
}

output "primary_host" {
  value = data.appdynamics_database_server.primary.host
}
