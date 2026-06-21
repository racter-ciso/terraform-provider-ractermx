data "ractermx_reputation_score" "example" {
  domain_id = ractermx_domain.example.id
}

output "reputation_grade" {
  value = data.ractermx_reputation_score.example.grade
}

output "reputation_score" {
  value = data.ractermx_reputation_score.example.composite_score
}

output "is_degraded" {
  value = data.ractermx_reputation_score.example.is_degraded
}
