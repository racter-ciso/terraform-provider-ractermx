# Read the health status of a domain's DNS configuration.
data "ractermx_domain_health" "example" {
  domain_id = ractermx_domain.example.id
}

output "overall_status" {
  value = data.ractermx_domain_health.example.overall_status
}

output "mx_status" {
  value = data.ractermx_domain_health.example.checks.mx.status
}
