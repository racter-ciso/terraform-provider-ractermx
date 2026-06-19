# Create an email alias that forwards info@example.com to a destination address.
resource "ractermx_alias" "info" {
  domain_id  = ractermx_domain.example.id
  local_part = "info"
  forward_to = "team@company.com"
}

# Create a catch-all alias that forwards all unmatched addresses.
resource "ractermx_alias" "catchall" {
  domain_id   = ractermx_domain.example.id
  is_catchall = true
  forward_to  = "admin@company.com"
  description = "Catch-all alias"
}

resource "ractermx_domain" "example" {
  name = "example.com"
}
