# Manage the email log retention policy.
# This is a singleton resource — only one per organization.
resource "ractermx_retention_policy" "default" {
  metadata_retention_days = 90

  event_specific_retention = {
    "bounced"  = 180
    "rejected" = 30
  }
}
