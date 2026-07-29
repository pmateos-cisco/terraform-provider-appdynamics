terraform {
  required_providers {
    appdynamics = {
      source = "pmateos-cisco/appdynamics"
    }
  }
}

provider "appdynamics" {
  controller_url = "https://mycompany.saas.appdynamics.com"
  client_id      = "my-api-client@mycompany" # <api_client_name>@<account_name>
  client_secret  = var.appdynamics_client_secret
}

variable "appdynamics_client_secret" {
  type      = string
  sensitive = true
}
