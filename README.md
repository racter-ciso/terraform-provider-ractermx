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
# RacterMX Demo Account Setup
# Provisions a multi-tenant org hierarchy with domains and aliases.
#
# Usage:
#   terraform init
#   terraform apply -var="api_key=sk_your_key_here"

terraform {
  required_providers {
    ractermx = {
      source  = "racter-ciso/ractermx"
      version = "~> 1.0"
    }
  }
}

variable "api_key" {
  type      = string
  sensitive = true
}

provider "ractermx" {
  api_key = var.api_key
}

# ─── Organizations ────────────────────────────────────────────────

data "ractermx_organizations" "account" {}

resource "ractermx_organization" "acme_corp" {
  name      = "ACME Corp"
  parent_id = data.ractermx_organizations.account.root_id
}

resource "ractermx_organization" "roadrunner" {
  name      = "RoadRunner LLC"
  parent_id = ractermx_organization.acme_corp.id
}

resource "ractermx_organization" "coyotetech" {
  name      = "CoyoteTech"
  parent_id = ractermx_organization.acme_corp.id
}

# ─── RoadRunner LLC Domains ───────────────────────────────────────

resource "ractermx_domain" "beepbeep" {
  name            = "*.BeepBeep.com"
  organization_id = ractermx_organization.roadrunner.id
  dns_mode        = "mx_forwarding"
  max_aliases     = 100
}

resource "ractermx_domain" "runningforfun" {
  name            = "RunningForFun.com"
  organization_id = ractermx_organization.roadrunner.id
  dns_mode        = "mx_forwarding"
  max_aliases     = 100
}

# ─── CoyoteTech Domains ──────────────────────────────────────────

resource "ractermx_domain" "acme" {
  name            = "Acme.com"
  organization_id = ractermx_organization.coyotetech.id
  dns_mode        = "mx_forwarding"
  max_aliases     = 200
}

resource "ractermx_domain" "howtopainttunnels" {
  name            = "HowToPaintTunnels.com"
  organization_id = ractermx_organization.coyotetech.id
  dns_mode        = "mx_forwarding"
  max_aliases     = 50
}

resource "ractermx_domain" "howbirdsthink" {
  name            = "howbirdsthink.com"
  organization_id = ractermx_organization.coyotetech.id
  dns_mode        = "mx_forwarding"
  max_aliases     = 50
}

# ─── Sample Aliases ───────────────────────────────────────────────

resource "ractermx_alias" "beepbeep_info" {
  domain_id  = ractermx_domain.beepbeep.id
  local_part = "info"
  forward_to = "roadrunner@gmail.com"
}

resource "ractermx_alias" "runningforfun_contact" {
  domain_id  = ractermx_domain.runningforfun.id
  local_part = "contact"
  forward_to = "speedy@roadrunner.io"
}

resource "ractermx_alias" "acme_sales" {
  domain_id  = ractermx_domain.acme.id
  local_part = "sales"
  forward_to = "wile.e.coyote@gmail.com"
}

resource "ractermx_alias" "tunnels_hello" {
  domain_id  = ractermx_domain.howtopainttunnels.id
  local_part = "hello"
  forward_to = "coyote@acme.com"
}

resource "ractermx_alias" "birds_research" {
  domain_id  = ractermx_domain.howbirdsthink.id
  local_part = "research"
  forward_to = "coyote@acme.com"
}

# ─── Outputs ──────────────────────────────────────────────────────

output "org_tree" {
  value = {
    acme_corp = {
      id = ractermx_organization.acme_corp.id
      children = {
        roadrunner = {
          id      = ractermx_organization.roadrunner.id
          domains = [ractermx_domain.beepbeep.name, ractermx_domain.runningforfun.name]
        }
        coyotetech = {
          id      = ractermx_organization.coyotetech.id
          domains = [ractermx_domain.acme.name, ractermx_domain.howtopainttunnels.name, ractermx_domain.howbirdsthink.name]
        }
      }
    }
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
