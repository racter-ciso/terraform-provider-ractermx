# Terraform Provider for RacterMX

Manage your [RacterMX](https://ractermx.com) email infrastructure as code — domains, aliases, DNS zones, security posture, DMARC, reputation monitoring, alerts, and more.

[![Terraform Registry](https://img.shields.io/badge/terraform-registry-blueviolet)](https://registry.terraform.io/providers/racter-ciso/ractermx/latest)

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.21 (to build the provider)
- A RacterMX API key (`sk_*`)

## Installation

```hcl
terraform {
  required_providers {
    ractermx = {
      source  = "racter-ciso/ractermx"
      version = "~> 0.1"
    }
  }
}

provider "ractermx" {
  api_key = var.ractermx_api_key
  # Or set RACTERMX_API_KEY environment variable
}
```

## Quick Start

```hcl
# Create a domain
resource "ractermx_domain" "example" {
  name       = "example.com"
  max_aliases = 100
}

# Verify DNS
resource "ractermx_domain_verification" "example" {
  domain_id = ractermx_domain.example.id
}

# Create an alias
resource "ractermx_alias" "info" {
  domain_id  = ractermx_domain.example.id
  local_part = "info"
  forward_to = "team@company.com"
}

# Create a DNS zone record
resource "ractermx_zone_record" "www" {
  domain_id = ractermx_domain.example.id
  name      = "www"
  type      = "CNAME"
  content   = "example.com"
  ttl       = 3600
}

# Monitor security posture
data "ractermx_security_score" "example" {
  domain_id = ractermx_domain.example.id
}

# Set up alerts
resource "ractermx_alert_rule" "blacklist" {
  domain_id  = ractermx_domain.example.id
  name       = "Blacklist Alert"
  alert_type = "blacklist_change"
  condition  = "any_change"

  channels {
    channel_type  = "email"
    email_address = "ops@company.com"
  }
}
```

## Resources

| Resource | Description |
|----------|-------------|
| `ractermx_domain` | Email forwarding domain |
| `ractermx_domain_verification` | DNS verification trigger |
| `ractermx_alias` | Email alias/forwarding rule |
| `ractermx_zone_record` | DNS zone record |
| `ractermx_webhook` | Webhook endpoint |
| `ractermx_blocklist_entry` | Sender blocklist entry |
| `ractermx_smtp_credential` | SMTP credential |
| `ractermx_api_key` | API key |
| `ractermx_retention_policy` | Email log retention settings |
| `ractermx_organization` | Organization management |
| `ractermx_domain_tag` | Domain tag |
| `ractermx_domain_tag_assignment` | Tag-to-domain assignment |
| `ractermx_alert_rule` | Alert rule |
| `ractermx_domain_notification_preference` | Per-domain notification prefs |
| `ractermx_check_override` | Security check override |

## Data Sources

| Data Source | Description |
|-------------|-------------|
| `ractermx_domain_dns_records` | Required DNS records for a domain |
| `ractermx_domain_statistics` | Email statistics for a domain |
| `ractermx_domain_health` | SPF/DKIM/DMARC/MX health status |
| `ractermx_quota` | Account quota and usage |
| `ractermx_security_score` | Security posture score |
| `ractermx_security_checks` | Security check results by pillar |
| `ractermx_check_catalog` | Available security checks |
| `ractermx_reputation_score` | Outbound email reputation |
| `ractermx_dmarc_compliance` | DMARC compliance rate |
| `ractermx_statistics` | Aggregated email statistics |

## Authentication

The provider uses a RacterMX API key for authentication. You can provide it in the provider block or via the `RACTERMX_API_KEY` environment variable:

```bash
export RACTERMX_API_KEY="sk_your_key_here"
```

## Documentation

Full documentation is available on the [Terraform Registry](https://registry.terraform.io/providers/racter-ciso/ractermx/latest/docs).

## License

MPL-2.0 — see [LICENSE](LICENSE).
