# Configure notification preferences for a domain.
resource "ractermx_domain_notification_preference" "example" {
  domain_id    = ractermx_domain.example.id
  muted        = false
  min_priority = "high"
}

resource "ractermx_domain" "example" {
  name = "example.com"
}
