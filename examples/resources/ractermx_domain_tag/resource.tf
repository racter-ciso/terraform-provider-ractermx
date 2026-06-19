# Manage a domain tag for organizing domains.
resource "ractermx_domain_tag" "production" {
  name  = "production"
  color = "#22c55e"
}

# Create a tag with the default color.
resource "ractermx_domain_tag" "staging" {
  name = "staging"
}
