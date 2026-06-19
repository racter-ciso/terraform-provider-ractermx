# Manage DNS zone records for a RacterMX-hosted domain.

# Create an A record pointing to a web server.
resource "ractermx_zone_record" "web" {
  domain_id = ractermx_domain.example.id
  name      = "www"
  type      = "A"
  content   = "203.0.113.10"
  ttl       = 3600
}

# Create an MX record for mail delivery.
resource "ractermx_zone_record" "mx" {
  domain_id = ractermx_domain.example.id
  name      = "@"
  type      = "MX"
  content   = "mail.example.com"
  ttl       = 3600
  priority  = 10
}

resource "ractermx_domain" "example" {
  name = "example.com"
}
