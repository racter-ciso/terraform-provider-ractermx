# Override a security check for a specific domain.
resource "ractermx_check_override" "disable_dmarc_strict" {
  domain_id         = ractermx_domain.example.id
  check_id          = "dmarc_strict_policy"
  enabled           = false
  severity_override = "low"
}

resource "ractermx_domain" "example" {
  name = "example.com"
}
