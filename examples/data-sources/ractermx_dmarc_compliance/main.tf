data "ractermx_dmarc_compliance" "example" {
  domain_id = ractermx_domain.example.id
}

output "dmarc_compliance_rate" {
  value = data.ractermx_dmarc_compliance.example.compliance_rate
}

output "dmarc_recommendation" {
  value = data.ractermx_dmarc_compliance.example.recommendation
}
