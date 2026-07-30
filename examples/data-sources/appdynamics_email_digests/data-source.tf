data "appdynamics_email_digests" "all" {
  application_id = 42
}

output "email_digest_names" {
  value = { for ed in data.appdynamics_email_digests.all.email_digests : ed.id => ed.name }
}
