# Read email statistics for a domain.
data "ractermx_domain_statistics" "example" {
  domain_id = ractermx_domain.example.id
  date_from = "2024-01-01"
  date_to   = "2024-01-31"
}

output "total_forwarded" {
  value = data.ractermx_domain_statistics.example.total_forwarded
}
