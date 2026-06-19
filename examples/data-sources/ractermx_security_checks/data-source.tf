# Read security findings for a domain.
data "ractermx_security_checks" "example" {
  domain_id = ractermx_domain.example.id
}

output "findings_count" {
  value = length(data.ractermx_security_checks.example.findings)
}
