# Create a webhook endpoint to receive email event notifications.
resource "ractermx_webhook" "events" {
  url    = "https://hooks.example.com/ractermx"
  events = ["delivered", "bounced", "failed"]

  custom_headers = {
    "X-Webhook-Source" = "ractermx"
  }

  timeout_seconds = 15
  batch_enabled   = true
}
