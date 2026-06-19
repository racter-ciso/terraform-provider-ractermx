# Read the required DNS records for a domain.
# Use these values to configure DNS at your DNS provider.
data "ractermx_domain_dns_records" "example" {
  domain_id = ractermx_domain.example.id
}

output "mx_record_value" {
  value = data.ractermx_domain_dns_records.example.mx.value
}

output "spf_record_value" {
  value = data.ractermx_domain_dns_records.example.spf.value
}
