# Changelog

## v0.1.2 (2026-06-19)

- Generated full provider documentation for the Terraform Registry
- Added README, LICENSE, and CHANGELOG

## v0.1.1 (2026-06-19)

- Added registry manifest file
- Fixed release config to publish (not draft)

## v0.1.0 (2026-06-19)

Initial release.

### Resources

- `ractermx_domain` — Email forwarding domain
- `ractermx_domain_verification` — DNS verification trigger
- `ractermx_alias` — Email alias / forwarding rule
- `ractermx_zone_record` — DNS zone record
- `ractermx_webhook` — Webhook endpoint
- `ractermx_blocklist_entry` — Sender blocklist entry
- `ractermx_smtp_credential` — SMTP credential
- `ractermx_api_key` — API key
- `ractermx_retention_policy` — Email log retention
- `ractermx_organization` — Organization management
- `ractermx_domain_tag` — Domain tag
- `ractermx_domain_tag_assignment` — Tag-to-domain assignment
- `ractermx_alert_rule` — Alert rule
- `ractermx_domain_notification_preference` — Per-domain notification prefs
- `ractermx_check_override` — Security check override

### Data Sources

- `ractermx_domain_dns_records`
- `ractermx_domain_statistics`
- `ractermx_domain_health`
- `ractermx_quota`
- `ractermx_security_score`
- `ractermx_security_checks`
- `ractermx_check_catalog`
- `ractermx_reputation_score`
- `ractermx_dmarc_compliance`
- `ractermx_statistics`
