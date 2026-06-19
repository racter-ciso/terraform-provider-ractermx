# Assign a tag to a domain.
resource "ractermx_domain_tag_assignment" "example" {
  domain_id = ractermx_domain.example.id
  tag_id    = ractermx_domain_tag.production.id
}

resource "ractermx_domain" "example" {
  name = "example.com"
}

resource "ractermx_domain_tag" "production" {
  name  = "production"
  color = "#22c55e"
}
