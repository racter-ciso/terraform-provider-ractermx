# Trigger DNS verification for a domain.
# To re-run verification, taint this resource and re-apply.
resource "ractermx_domain_verification" "example" {
  domain_id = ractermx_domain.example.id
}

resource "ractermx_domain" "example" {
  name = "example.com"
}
