# Read the security posture score for a domain.
data "ractermx_security_score" "example" {
  domain_id = ractermx_domain.example.id
}

output "security_grade" {
  value = data.ractermx_security_score.example.grade
}

output "overall_score" {
  value = data.ractermx_security_score.example.overall_score
}
