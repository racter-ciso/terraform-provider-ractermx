# Create an alert rule that triggers when the security posture score drops below B.
resource "ractermx_alert_rule" "security_alert" {
  domain_id       = ractermx_domain.example.id
  name            = "Security Score Alert"
  alert_type      = "security_posture"
  condition       = "below"
  threshold_value = "B"
  cooldown_minutes = 120

  channels {
    channel_type = "email"
    email_address = "ops@example.com"
  }
}

# Create an alert rule for blacklist changes with a webhook notification.
resource "ractermx_alert_rule" "blacklist_alert" {
  domain_id  = ractermx_domain.example.id
  name       = "Blacklist Change Alert"
  alert_type = "blacklist_change"
  condition  = "any_change"

  channels {
    channel_type        = "webhook"
    webhook_endpoint_id = ractermx_webhook.events.id
  }
}

resource "ractermx_domain" "example" {
  name = "example.com"
}

resource "ractermx_webhook" "events" {
  url    = "https://hooks.example.com/ractermx"
  events = ["delivered", "bounced"]
}
