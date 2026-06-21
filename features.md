# RacterMX Terraform Provider — Feature List

> Infrastructure-as-Code for Email Forwarding  
> Source: `ractermx-terraform/`

---

## Overview

- 15 managed resources
- 10 read-only data sources
- 25 total Terraform types
- Full documentation for all resources and data sources
- Example configurations for all resources
- Import support on all resources (except domain_verification)
- Proper 404 handling on all read operations (removes from state when deleted externally)
- Sensitive field marking on all credentials (API keys, SMTP passwords, webhook secrets)

---

## Resources

### ractermx_domain

- Create, read, update, and delete email forwarding domains
- Configure: organization_id, dns_mode, catch-all forwarding, max aliases, monitoring, hosting
- Computed outputs: verification status (MX, SPF, DKIM, DMARC), verification token, timestamps
- Import by numeric ID

### ractermx_alias

- Create, read, update, and delete email aliases per domain
- Configure: domain_id, local_part, forward_to, is_catchall, description
- Computed outputs: is_active, is_wildcard, timestamps
- Import by numeric ID

### ractermx_api_key

- Create and revoke API keys
- Configure: name, scopes (15+ granular permissions), optional expiration, IP allowlist
- Computed outputs: api_key (sensitive), last_used_at, timestamps
- Immutable — any change triggers destroy+create
- Import by numeric ID

### ractermx_blocklist_entry

- Create and delete sender blocklist entries
- Configure: pattern (exact address or wildcard, e.g., `*@spam.com`)
- Immutable — pattern change triggers destroy+create
- Import by numeric ID

### ractermx_webhook

- Create, update, and delete webhook endpoints
- Configure: url, events, custom_headers, timeout_seconds, batch_enabled, enabled
- Computed outputs: secret (sensitive), timestamps
- Full event type support (11 event types)
- Import by numeric ID

### ractermx_zone_record

- Create, update, and delete DNS zone records
- Configure: domain_id, name, type, content, ttl, priority, weight, port
- Supports all record types: A, AAAA, CNAME, MX, TXT, SRV, NS, CAA
- Import by composite ID: `{domain_id}/{name}/{type}/{content}`

### ractermx_smtp_credential

- Create and delete SMTP credentials per domain
- Configure: domain_id, daily_limit, anonymous_reply_enabled, proxy_domain_id
- Computed outputs: username, password (sensitive), smtp_config (host, port, encryption)
- Immutable — any change triggers destroy+create
- Import by numeric ID

### ractermx_organization

- Create, update, and delete organizations in a hierarchical tree
- Configure: name, parent_id
- Computed outputs: users_count, domains_count, total_domains_count
- Import by numeric ID

### ractermx_alert_rule

- Create, update, and delete monitoring alert rules
- Configure: domain_id, name, alert_type, condition, threshold_value, cooldown_minutes, enabled, channels
- Alert types: deliverability_score, blacklist_change, security_posture, dmarc_compliance
- Multi-channel notifications (email + webhook)
- Import by numeric ID

### ractermx_check_override

- Create and update per-domain security check overrides
- Configure: domain_id, check_id, enabled, severity_override
- Override severity: critical, high, medium, low, informational
- Delete resets to defaults
- Import by composite ID: `{domain_id}/{check_id}`

### ractermx_domain_tag

- Create, update, and delete domain tags
- Configure: name, color (hex)
- Computed outputs: domains_count
- Import by numeric ID

### ractermx_domain_tag_assignment

- Assign tags to domains
- Configure: domain_id, tag_id
- Immutable — both attributes are identity fields, change triggers replace
- Import by composite ID: `{domain_id}/{tag_id}`

### ractermx_domain_verification

- Trigger DNS verification for a domain
- Computed outputs: mx_verified, spf_verified, dkim_verified, dmarc_verified, is_verified
- Create triggers verification; Delete is a no-op (verification cannot be undone)
- No import support

### ractermx_domain_notification_preference

- Configure per-domain notification preferences
- Configure: domain_id, muted, min_priority
- Delete resets to defaults
- Import by domain ID

### ractermx_retention_policy

- Configure email log retention policy (singleton resource)
- Configure: metadata_retention_days (7–2555), event_specific_retention (per-event overrides)
- Delete removes from state only (singleton cannot be deleted)
- Import with ID "default"

---

## Data Sources

### ractermx_check_catalog

- Read the full security check catalog grouped by pillar
- Returns: check_id, pillar, name, description, default_severity, version

### ractermx_dmarc_compliance

- Read DMARC compliance rate for a domain
- Returns: compliance_rate, total_messages, passed_messages, current_policy, recommendation

### ractermx_domain_dns_records

- Read required DNS records for a domain (what to configure at registrar)
- Returns: mx, spf, dkim, dmarc records (type, name, value, ttl)

### ractermx_domain_health

- Read domain health dashboard
- Returns: overall_status, domain_verified, per-check status (mx, spf, dkim, dmarc)

### ractermx_domain_statistics

- Read email statistics for a domain with optional date range
- Returns: total_received, total_forwarded, total_bounced, total_deferred, total_rejected

### ractermx_quota

- Read account quota and usage
- Returns: domains_limit/used, aliases_limit/used, smtp_credentials_limit/used

### ractermx_reputation_score

- Read composite outbound email reputation score
- Returns: composite_score, grade, is_degraded, insufficient_data, total_sent, computed_at

### ractermx_security_checks

- Read all security check findings for a domain
- Returns: findings[] with check_id, pillar, name, description, status, severity, fix_available

### ractermx_security_score

- Read security posture score with pillar breakdown
- Returns: overall_score, grade, pillars[] (name, score, grade)

### ractermx_statistics

- Read aggregated email statistics with optional date range
- Returns: total_received, total_forwarded, total_bounced, total_deferred, total_rejected, total_bytes

---

## Provider Characteristics

- Go-based provider using Terraform Plugin Framework
- Bearer token authentication via API key
- Configurable base URL (supports self-hosted deployments)
- All read operations handle 404 gracefully (remove from state)
- Composite resource identification for DNS records, tag assignments, and check overrides
- Full acceptance test suite
- Published documentation site
