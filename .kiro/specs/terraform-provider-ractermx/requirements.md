# Requirements Document

## Introduction

This document defines the requirements for a Terraform provider for RacterMX, an email forwarding and mail routing management platform. The provider wraps the RacterMX REST API v2 (`/api/v2/`), enabling infrastructure-as-code management of email domains, aliases, DNS zone records, webhooks, blocklist entries, SMTP credentials, API keys, retention policies, organizations, domain tags, alert rules, notification preferences, and check catalog overrides.

The provider is written in Go using the HashiCorp Terraform Plugin Framework (not the legacy SDK). It authenticates via Bearer token (`sk_...`) and targets a configurable base URL (default `https://ractermx.com`). The implementation follows a phased approach given the XL effort estimate.

## Glossary

- **Provider**: The top-level Terraform provider binary that authenticates with the RacterMX API and exposes resources and data sources.
- **API_Client**: The internal Go HTTP client that sends authenticated requests to the RacterMX API v2 and parses JSON responses.
- **Resource**: A Terraform resource representing a CRUD-managed object in RacterMX (e.g., `ractermx_domain`, `ractermx_alias`).
- **Data_Source**: A Terraform data source representing a read-only query against the RacterMX API (e.g., `ractermx_domain_health`, `ractermx_domain_statistics`).
- **Plugin_Framework**: The HashiCorp `terraform-plugin-framework` Go module used to build modern Terraform providers.
- **Acceptance_Test**: An end-to-end test that creates, reads, updates, and destroys real RacterMX resources via the API during `terraform apply`.
- **Domain**: An email domain registered in RacterMX for forwarding, monitoring, or DNS hosting.
- **Alias**: An email forwarding rule mapping a local part (e.g., `info`) on a Domain to a destination email address.
- **Zone_Record**: A DNS record in a domain's hosted zone (only available for domains with `dns_mode = dns_hosted`).
- **Webhook_Endpoint**: A URL that receives event notifications from RacterMX via signed HTTP POST requests.
- **Blocklist_Entry**: A sender email pattern (exact address or wildcard like `*@spam.com`) that blocks incoming mail.
- **SMTP_Credential**: Authentication credentials for sending email through RacterMX's SMTP relay, scoped to a Domain.
- **API_Key**: A bearer token with scoped permissions and optional IP allowlisting for authenticating with the RacterMX API.
- **Retention_Policy**: A singleton per-organization configuration controlling how long email metadata and event-specific logs are retained.
- **Organization**: A hierarchical grouping unit within a tenant for organizing domains and users.
- **Domain_Tag**: A label with a name and color that can be assigned to domains for filtering and grouping.
- **Alert_Rule**: A monitoring rule that fires notifications when domain metrics cross configured thresholds.
- **Notification_Preference**: Per-domain settings controlling whether notifications are muted and the minimum priority threshold.
- **Check_Override**: A per-domain override for a security check catalog entry, allowing checks to be disabled or severity adjusted.
- **HCL**: HashiCorp Configuration Language, the declarative syntax used in Terraform configuration files.
- **tfplugindocs**: The HashiCorp tool that generates provider documentation from schema annotations and templates.

## Requirements

### Requirement 1: Provider Configuration and Authentication

**User Story:** As a DevOps engineer, I want to configure the RacterMX Terraform provider with my API key and optional base URL, so that I can authenticate all resource operations against my RacterMX account.

#### Acceptance Criteria

1. THE Provider SHALL accept an `api_key` attribute in the provider configuration block.
2. WHEN the `api_key` attribute is not set in HCL, THE Provider SHALL read the `RACTERMX_API_KEY` environment variable as a fallback.
3. IF neither the `api_key` attribute nor the `RACTERMX_API_KEY` environment variable is set, THEN THE Provider SHALL return a diagnostic error during configuration and refuse to proceed.
4. THE Provider SHALL accept an optional `base_url` attribute that defaults to `https://ractermx.com`.
5. THE API_Client SHALL include the configured API key as a `Bearer` token in the `Authorization` header of every HTTP request.
6. THE API_Client SHALL append `/api/v2` to the configured base URL for all API requests.
7. IF the API returns a 401 Unauthorized response, THEN THE API_Client SHALL return a descriptive error indicating invalid or expired credentials.
8. THE Provider SHALL mark the `api_key` attribute as sensitive so that Terraform does not display the value in plan output or state logs.

### Requirement 2: API Client HTTP Handling

**User Story:** As a DevOps engineer, I want the provider to handle API errors, rate limits, and pagination gracefully, so that Terraform operations are reliable and predictable.

#### Acceptance Criteria

1. WHEN the API returns a 422 response with an `errors` object, THE API_Client SHALL parse the validation errors and return them as Terraform diagnostic messages.
2. WHEN the API returns a 429 Too Many Requests response, THE API_Client SHALL retry the request with exponential backoff up to 3 times.
3. WHEN the API returns a 404 response during a Read operation, THE API_Client SHALL remove the resource from Terraform state (signal that the resource was deleted out-of-band).
4. WHEN the API returns a 5xx response, THE API_Client SHALL retry the request up to 2 times with exponential backoff before returning an error.
5. THE API_Client SHALL set a configurable request timeout defaulting to 30 seconds.
6. THE API_Client SHALL include a `User-Agent` header in the format `terraform-provider-ractermx/<version>`.
7. WHEN the API returns paginated results with a `meta` object, THE API_Client SHALL iterate through all pages to return the complete result set.

### Requirement 3: Domain Resource

**User Story:** As a DevOps engineer, I want to manage RacterMX domains as Terraform resources, so that I can version-control domain configuration and automate provisioning.

#### Acceptance Criteria

1. THE Resource `ractermx_domain` SHALL support Create, Read, Update, and Delete operations mapped to `POST /domains`, `GET /domains/{id}`, `PATCH /domains/{id}`, and `DELETE /domains/{id}`.
2. THE Resource SHALL require the `name` attribute (string, must match `^[a-z0-9.-]+$`).
3. THE Resource SHALL accept optional attributes: `organization_id` (integer), `is_forwarding` (boolean), `is_monitored` (boolean), `is_hosted` (boolean), `dns_mode` (string, one of `scan_only`, `mx_forwarding`, `dns_hosted`), `catch_all_enabled` (boolean), `catch_all_forward_to` (string), and `max_aliases` (integer, 1–1000, default 100).
4. THE Resource SHALL expose computed read-only attributes: `id`, `is_active`, `is_verified`, `verification_token`, `mx_verified`, `spf_verified`, `dkim_verified`, `dmarc_verified`, `last_verified_at`, `created_at`, and `updated_at`.
5. WHEN the `name` attribute is changed, THE Resource SHALL force replacement (destroy and recreate) because domain names are immutable after creation.
6. THE Resource SHALL support the Terraform import operation using the domain's numeric ID.
7. WHEN a domain is deleted, THE Resource SHALL also delete all associated aliases as the API does.

### Requirement 4: Domain DNS Verification Action

**User Story:** As a DevOps engineer, I want to trigger DNS verification for a domain after configuring DNS records, so that I can automate the domain verification workflow.

#### Acceptance Criteria

1. THE Resource `ractermx_domain_verification` SHALL trigger a `POST /domains/{id}/verify-dns` call on Create.
2. THE Resource SHALL expose computed attributes: `mx_verified`, `spf_verified`, `dkim_verified`, `dmarc_verified`, and `is_verified`.
3. WHEN any verification result is false, THE Resource SHALL not return an error but SHALL expose the individual verification statuses so that downstream resources or outputs can reference them.
4. THE Resource SHALL support re-running verification by tainting and re-applying.

### Requirement 5: Alias Resource

**User Story:** As a DevOps engineer, I want to manage email aliases as Terraform resources, so that I can declaratively define forwarding rules for each domain.

#### Acceptance Criteria

1. THE Resource `ractermx_alias` SHALL support Create, Read, Update, and Delete operations mapped to `POST /domains/{domainId}/aliases`, `GET /aliases/{id}`, `PATCH /aliases/{id}`, and `DELETE /aliases/{id}`.
2. THE Resource SHALL require `domain_id` (integer) and `forward_to` (string, valid email).
3. THE Resource SHALL accept optional attributes: `local_part` (string, max 64 chars, regex `^[a-z0-9._+-]+$`), `is_catchall` (boolean, defaults to false), and `description` (string, max 255 chars).
4. WHEN `is_catchall` is true, THE Resource SHALL set `local_part` to `*` regardless of any user-provided value.
5. THE Resource SHALL expose computed read-only attributes: `id`, `is_active`, `is_wildcard`, `created_at`, and `updated_at`.
6. WHEN the `local_part` or `domain_id` attribute is changed, THE Resource SHALL force replacement because alias identity is immutable.
7. IF the API returns a 409 Conflict (duplicate alias), THEN THE Resource SHALL return a clear error message including the full alias address.
8. THE Resource SHALL support the Terraform import operation using the alias numeric ID.

### Requirement 6: DNS Zone Record Resource

**User Story:** As a DevOps engineer, I want to manage DNS zone records for RacterMX-hosted domains, so that I can version-control DNS configuration alongside email forwarding rules.

#### Acceptance Criteria

1. THE Resource `ractermx_zone_record` SHALL support Create, Read, Update, and Delete operations mapped to `POST /domains/{domainId}/zone-records`, `GET /domains/{domainId}/zone-records` (list and match), `PATCH /domains/{domainId}/zone-records`, and `DELETE /domains/{domainId}/zone-records`.
2. THE Resource SHALL require `domain_id` (integer), `name` (string), `type` (string), `content` (string), and `ttl` (integer, 60–86400).
3. THE Resource SHALL accept optional attributes: `priority` (integer, 0–65535, for MX/SRV), `weight` (integer, 0–65535, for SRV), and `port` (integer, 1–65535, for SRV).
4. IF the domain's `dns_mode` is not `dns_hosted`, THEN THE Resource SHALL return an error explaining that zone records require DNS hosting.
5. WHEN updating a zone record, THE API_Client SHALL send the old record values (name, type, content) and new record values as required by the API's old/new update pattern.
6. THE Resource SHALL use a composite identifier of `domain_id`, `name`, `type`, and `content` for Read and Delete operations since zone records have no server-assigned ID.
7. THE Resource SHALL support the Terraform import operation using a composite key in the format `{domain_id}/{name}/{type}/{content}`.

### Requirement 7: Webhook Endpoint Resource

**User Story:** As a DevOps engineer, I want to manage webhook endpoints as Terraform resources, so that I can declaratively configure event delivery for monitoring and integration pipelines.

#### Acceptance Criteria

1. THE Resource `ractermx_webhook` SHALL support Create, Read, Update, and Delete operations mapped to `POST /webhooks`, `GET /webhooks` (list and match by ID), `PUT /webhooks/{id}`, and `DELETE /webhooks/{id}`.
2. THE Resource SHALL require `url` (string, valid URL, max 2048 chars) and `events` (list of strings from the valid event type enum).
3. THE Resource SHALL accept optional attributes: `custom_headers` (map of string to string), `timeout_seconds` (integer, 5–30, default 10), `batch_enabled` (boolean, default false), and `enabled` (boolean, default true).
4. THE Resource SHALL expose computed read-only attributes: `id`, `secret` (sensitive, only available on create), and `created_at`.
5. THE Resource SHALL mark the `secret` attribute as sensitive.
6. WHEN the webhook is created, THE Resource SHALL store the signing secret in state because the API only returns the secret on creation.
7. THE Resource SHALL support the Terraform import operation using the webhook numeric ID, noting that the `secret` attribute will not be populated on import.

### Requirement 8: Blocklist Entry Resource

**User Story:** As a DevOps engineer, I want to manage sender blocklist entries as Terraform resources, so that I can enforce email filtering policies through code.

#### Acceptance Criteria

1. THE Resource `ractermx_blocklist_entry` SHALL support Create, Read, and Delete operations mapped to `POST /blocklist`, `GET /blocklist` (list and match), and `DELETE /blocklist/{id}`.
2. THE Resource SHALL require the `pattern` attribute (string, max 255 chars, e.g., `spam@example.com` or `*@spam.com`).
3. THE Resource SHALL expose computed read-only attributes: `id` and `created_at`.
4. WHEN the `pattern` attribute is changed, THE Resource SHALL force replacement because blocklist entries have no update endpoint.
5. IF the API returns a 409 Conflict (pattern already exists), THEN THE Resource SHALL return a clear error message.
6. THE Resource SHALL support the Terraform import operation using the blocklist entry numeric ID.

### Requirement 9: SMTP Credential Resource

**User Story:** As a DevOps engineer, I want to manage SMTP credentials as Terraform resources, so that I can provision sending credentials alongside domain configuration.

#### Acceptance Criteria

1. THE Resource `ractermx_smtp_credential` SHALL support Create, Read, and Delete operations mapped to `POST /domains/{domainId}/smtp-credentials`, `GET /domains/{domainId}/smtp-credentials` (list and match), and `DELETE /smtp-credentials/{id}`.
2. THE Resource SHALL require `domain_id` (integer).
3. THE Resource SHALL accept optional attributes: `daily_limit` (integer, 1–100000, default 1000), `anonymous_reply_enabled` (boolean), and `proxy_domain_id` (integer).
4. THE Resource SHALL expose computed read-only attributes: `id`, `username`, `password` (sensitive, only on create), and `smtp_config` (object with host, port, encryption).
5. THE Resource SHALL mark the `password` attribute as sensitive.
6. WHEN the credential is created, THE Resource SHALL store the password in state because the API only returns the password on creation.
7. WHEN any attribute other than `domain_id` is changed, THE Resource SHALL force replacement because SMTP credentials have no general update endpoint.
8. THE Resource SHALL support the Terraform import operation using the credential numeric ID, noting that the `password` attribute will not be populated on import.

### Requirement 10: API Key Resource

**User Story:** As a DevOps engineer, I want to manage RacterMX API keys as Terraform resources, so that I can provision scoped access tokens for CI/CD pipelines and integrations.

#### Acceptance Criteria

1. THE Resource `ractermx_api_key` SHALL support Create, Read, and Delete operations mapped to `POST /api-keys`, `GET /api-keys` (list and match), and `DELETE /api-keys/{id}`.
2. THE Resource SHALL require `name` (string, max 255 chars) and `scopes` (list of strings from the valid scopes enum: `email:read`, `email:send`, `domains:read`, `domains:manage`, `aliases:read`, `aliases:manage`, `smtp:read`, `smtp:manage`, `webhooks:read`, `webhooks:manage`, `blocklist:read`, `blocklist:manage`, `api-keys:manage`, `retention:read`, `retention:manage`).
3. THE Resource SHALL accept optional attributes: `expires_at` (string, ISO 8601 datetime, must be in the future) and `allowed_ips` (list of strings, max 20 entries, valid IPv4/IPv6 addresses or CIDR blocks).
4. THE Resource SHALL expose computed read-only attributes: `id`, `api_key` (sensitive, only on create), `last_used_at`, and `created_at`.
5. THE Resource SHALL mark the `api_key` attribute as sensitive.
6. WHEN the key is created, THE Resource SHALL store the raw API key value in state because the API only returns the key on creation.
7. WHEN any attribute is changed, THE Resource SHALL force replacement because API keys have no update endpoint.
8. THE Resource SHALL support the Terraform import operation using the API key numeric ID, noting that the `api_key` attribute will not be populated on import.

### Requirement 11: Retention Policy Resource

**User Story:** As a DevOps engineer, I want to manage the email log retention policy as a Terraform resource, so that I can enforce compliance retention periods through code.

#### Acceptance Criteria

1. THE Resource `ractermx_retention_policy` SHALL support Read and Update operations mapped to `GET /retention-policy` and `PUT /retention-policy`.
2. THE Resource SHALL require `metadata_retention_days` (integer, 7–2555).
3. THE Resource SHALL accept an optional `event_specific_retention` attribute (map of string to integer, where each value is 7–2555).
4. THE Resource SHALL expose a computed `updated_at` attribute.
5. THE Resource SHALL be a singleton (one per organization) and SHALL NOT support Delete — removing the resource from configuration SHALL leave the policy unchanged on the server.
6. THE Resource SHALL support the Terraform import operation using a fixed identifier string `"default"` since there is only one retention policy per organization.

### Requirement 12: Organization Resource

**User Story:** As a DevOps engineer, I want to manage RacterMX organizations as Terraform resources, so that I can define the organizational hierarchy for multi-team domain management.

#### Acceptance Criteria

1. THE Resource `ractermx_organization` SHALL support Create, Read, Update, and Delete operations mapped to `POST /organizations`, `GET /organizations` (tree traversal to find by ID), `PATCH /organizations/{id}`, and `DELETE /organizations/{id}`.
2. THE Resource SHALL require `name` (string, max 255 chars) and `parent_id` (integer).
3. THE Resource SHALL expose computed read-only attributes: `id`, `users_count`, `domains_count`, and `total_domains_count`.
4. IF the organization has domains, child organizations, or non-self members, THEN THE Resource SHALL return the API's error message explaining what must be removed first.
5. THE Resource SHALL support the Terraform import operation using the organization numeric ID.
6. THE Resource SHALL NOT allow deletion of the user's primary organization, returning the API's error message.

### Requirement 13: Domain Tag Resource

**User Story:** As a DevOps engineer, I want to manage domain tags as Terraform resources, so that I can organize domains by purpose and apply consistent labeling.

#### Acceptance Criteria

1. THE Resource `ractermx_domain_tag` SHALL support Create, Read, Update, and Delete operations mapped to `POST /tags`, `GET /tags` (list and match), `PATCH /tags/{id}`, and `DELETE /tags/{id}`.
2. THE Resource SHALL require `name` (string, max 50 chars).
3. THE Resource SHALL accept an optional `color` attribute (string, hex format `#RRGGBB`, default `#3b82f6`).
4. THE Resource SHALL expose computed read-only attributes: `id` and `domains_count`.
5. IF a tag with the same name already exists, THEN THE Resource SHALL return a clear error message.
6. THE Resource SHALL support the Terraform import operation using the tag numeric ID.

### Requirement 14: Domain Tag Assignment Resource

**User Story:** As a DevOps engineer, I want to assign tags to domains as Terraform resources, so that I can declaratively manage domain categorization.

#### Acceptance Criteria

1. THE Resource `ractermx_domain_tag_assignment` SHALL support Create and Delete operations mapped to `POST /domains/{id}/tags` and `DELETE /domains/{id}/tags/{tagId}`.
2. THE Resource SHALL require `domain_id` (integer) and `tag_id` (integer).
3. THE Resource SHALL use a composite identifier of `domain_id` and `tag_id`.
4. THE Resource SHALL support the Terraform import operation using a composite key in the format `{domain_id}/{tag_id}`.

### Requirement 15: Alert Rule Resource

**User Story:** As a DevOps engineer, I want to manage alert rules as Terraform resources, so that I can define monitoring thresholds and notification channels through code.

#### Acceptance Criteria

1. THE Resource `ractermx_alert_rule` SHALL support Create, Read, Update, and Delete operations mapped to `POST /alert-rules`, `GET /alert-rules/{id}`, `PATCH /alert-rules/{id}`, and `DELETE /alert-rules/{id}`.
2. THE Resource SHALL require `domain_id` (integer), `name` (string, 1–100 chars), `alert_type` (string, one of `deliverability_score`, `blacklist_change`, `security_posture`, `dmarc_compliance`), `condition` (string, one of `below`, `above`, `equals`, `any_change`), and `channels` (list of channel objects, 1–3 items).
3. THE Resource SHALL accept optional attributes: `threshold_value` (string, max 50 chars), `cooldown_minutes` (integer, 15–1440, default 60), and `enabled` (boolean, default true).
4. WHEN `alert_type` is `blacklist_change`, THE Resource SHALL validate that `condition` is `any_change` and `threshold_value` is null.
5. WHEN `alert_type` is `deliverability_score` or `security_posture`, THE Resource SHALL validate that `threshold_value` is a valid grade letter (A, B, C, D, F) and `condition` is not `any_change`.
6. WHEN `alert_type` is `dmarc_compliance`, THE Resource SHALL validate that `threshold_value` is an integer between 0 and 100 and `condition` is not `any_change`.
7. EACH channel object SHALL contain `channel_type` (string, `webhook` or `email`), and conditionally `webhook_endpoint_id` (integer, required when `channel_type` is `webhook`) or `email_address` (string, required when `channel_type` is `email`).
8. THE Resource SHALL expose computed read-only attributes: `id` and `created_at`.
9. THE Resource SHALL support the Terraform import operation using the alert rule numeric ID.

### Requirement 16: Domain Notification Preference Resource

**User Story:** As a DevOps engineer, I want to manage per-domain notification preferences as Terraform resources, so that I can control notification behavior for high-volume domains.

#### Acceptance Criteria

1. THE Resource `ractermx_domain_notification_preference` SHALL support Create (upsert), Read, and Delete operations mapped to `POST /domains/{id}/notification-preferences`, `GET /domains/{id}/notification-preferences`, and a logical delete that resets to defaults.
2. THE Resource SHALL require `domain_id` (integer).
3. THE Resource SHALL accept optional attributes: `muted` (boolean, default false) and `min_priority` (string, from the valid priorities enum).
4. WHEN the resource is removed from configuration, THE Resource SHALL reset the preference to defaults (unmuted, normal priority) rather than deleting a server-side record.
5. THE Resource SHALL support the Terraform import operation using the domain numeric ID.

### Requirement 17: Check Catalog Override Resource

**User Story:** As a DevOps engineer, I want to manage security check overrides per domain as Terraform resources, so that I can customize security scanning behavior for specific domains.

#### Acceptance Criteria

1. THE Resource `ractermx_check_override` SHALL support Create (upsert), Read, and Delete operations mapped to `PUT /domains/{id}/check-overrides/{checkId}` and `GET /check-catalog` (to verify check existence).
2. THE Resource SHALL require `domain_id` (integer) and `check_id` (string).
3. THE Resource SHALL accept optional attributes: `enabled` (boolean) and `severity_override` (string, one of `critical`, `high`, `medium`, `low`, `informational`).
4. WHEN the resource is removed from configuration, THE Resource SHALL send a request with null values to reset the override to catalog defaults.
5. THE Resource SHALL use a composite identifier of `domain_id` and `check_id`.
6. THE Resource SHALL support the Terraform import operation using a composite key in the format `{domain_id}/{check_id}`.

### Requirement 18: Domain DNS Records Data Source

**User Story:** As a DevOps engineer, I want to read the required DNS records for a domain, so that I can reference MX, SPF, DKIM, and DMARC values in other Terraform resources (e.g., a DNS provider).

#### Acceptance Criteria

1. THE Data_Source `ractermx_domain_dns_records` SHALL read from `GET /domains/{id}/dns-records`.
2. THE Data_Source SHALL require `domain_id` (integer).
3. THE Data_Source SHALL expose attributes for each record type: `mx`, `spf`, `dkim`, and `dmarc`, each containing `type`, `name`, `value`, and `ttl`.

### Requirement 19: Domain Statistics Data Source

**User Story:** As a DevOps engineer, I want to read email statistics for a domain, so that I can use forwarding metrics in monitoring dashboards or conditional logic.

#### Acceptance Criteria

1. THE Data_Source `ractermx_domain_statistics` SHALL read from `GET /domains/{id}/statistics`.
2. THE Data_Source SHALL require `domain_id` (integer).
3. THE Data_Source SHALL accept optional `date_from` and `date_to` attributes (strings, date format).
4. THE Data_Source SHALL expose attributes: `total_received`, `total_forwarded`, `total_bounced`, `total_deferred`, and `total_rejected`.

### Requirement 20: Domain Health Data Source

**User Story:** As a DevOps engineer, I want to read the health status of a domain's DNS configuration, so that I can gate downstream operations on verification status.

#### Acceptance Criteria

1. THE Data_Source `ractermx_domain_health` SHALL read from `GET /domains/{id}/health`.
2. THE Data_Source SHALL require `domain_id` (integer).
3. THE Data_Source SHALL expose attributes: `overall_status` (string: `pass`, `warning`, or `fail`), `domain_verified` (boolean), and a `checks` object containing `mx`, `spf`, `dkim`, and `dmarc` sub-objects each with `status`, `message`, and `checked_at`.

### Requirement 21: Security Score Data Source

**User Story:** As a DevOps engineer, I want to read the security posture score for a domain, so that I can enforce minimum security grades as policy.

#### Acceptance Criteria

1. THE Data_Source `ractermx_security_score` SHALL read from `GET /domains/{id}/security/score`.
2. THE Data_Source SHALL require `domain_id` (integer).
3. THE Data_Source SHALL expose attributes: `overall_score` (integer), `grade` (string), and pillar breakdown attributes.

### Requirement 22: Security Checks Data Source

**User Story:** As a DevOps engineer, I want to read security findings for a domain, so that I can audit security posture and identify fixable issues.

#### Acceptance Criteria

1. THE Data_Source `ractermx_security_checks` SHALL read from `GET /domains/{id}/security`.
2. THE Data_Source SHALL require `domain_id` (integer).
3. THE Data_Source SHALL expose a list of findings grouped by pillar, each containing check details, status, severity, and whether a fix is available.

### Requirement 23: Check Catalog Data Source

**User Story:** As a DevOps engineer, I want to read the available security check catalog, so that I can reference valid check IDs when configuring check overrides.

#### Acceptance Criteria

1. THE Data_Source `ractermx_check_catalog` SHALL read from `GET /check-catalog`.
2. THE Data_Source SHALL expose a list of checks grouped by pillar, each containing `check_id`, `name`, `description`, `default_severity`, and `version`.

### Requirement 24: Quota Data Source

**User Story:** As a DevOps engineer, I want to read account quota information, so that I can monitor resource usage and plan capacity.

#### Acceptance Criteria

1. THE Data_Source `ractermx_quota` SHALL read from `GET /quota`.
2. THE Data_Source SHALL expose quota attributes returned by the API.

### Requirement 25: Provider Scaffold and Build System

**User Story:** As a DevOps engineer, I want the provider to follow HashiCorp's standard project structure and build conventions, so that it is maintainable and publishable to the Terraform Registry.

#### Acceptance Criteria

1. THE Provider project SHALL follow the standard directory layout: `internal/provider/` for provider code, `internal/client/` for the API client, `internal/resources/` for resource implementations, `internal/datasources/` for data source implementations, and `docs/` for generated documentation.
2. THE Provider SHALL use Go modules with `go.mod` declaring `terraform-provider-ractermx` as the module name.
3. THE Provider SHALL include a `GNUmakefile` with targets for `build`, `install`, `test`, `testacc` (acceptance tests), and `generate` (documentation generation).
4. THE Provider SHALL include a `.goreleaser.yml` configuration for cross-platform binary builds and GitHub release automation.
5. THE Provider SHALL generate documentation using `tfplugindocs` from schema annotations and example templates in `examples/`.
6. THE Provider SHALL include a `main.go` entry point that registers the provider with the Plugin Framework's `providerserver`.

### Requirement 26: Acceptance Testing

**User Story:** As a provider developer, I want comprehensive acceptance tests for each resource and data source, so that I can verify correct behavior against the real RacterMX API.

#### Acceptance Criteria

1. EACH Resource SHALL have acceptance tests covering the full lifecycle: Create, Read, Update (where applicable), and Delete.
2. EACH Data_Source SHALL have acceptance tests verifying that attributes are populated correctly.
3. THE Acceptance_Test suite SHALL use the `terraform-plugin-testing` framework with `resource.Test` and `resource.TestStep`.
4. THE Acceptance_Test suite SHALL read API credentials from the `RACTERMX_API_KEY` environment variable and skip tests when the variable is not set.
5. EACH acceptance test SHALL use a `CheckDestroy` function that verifies the resource was deleted from RacterMX after the test completes.
6. THE Acceptance_Test suite SHALL use randomized resource names (e.g., `acctest-<random>.example.com` for domains) to avoid collisions between parallel test runs.

### Requirement 27: Phased Delivery

**User Story:** As a project stakeholder, I want the provider delivered in phases, so that core value is available early while the full API surface is covered incrementally.

#### Acceptance Criteria

1. THE Provider SHALL be delivered in four phases:
   - Phase 1: Provider scaffold, API client, Domain resource, Alias resource, and their data sources.
   - Phase 2: Zone Record resource, Webhook resource, Blocklist Entry resource.
   - Phase 3: SMTP Credential resource, API Key resource, Retention Policy resource, Organization resource.
   - Phase 4: Alert Rule resource, Domain Tag resource, Domain Tag Assignment resource, Notification Preference resource, Check Override resource, and remaining data sources.
2. EACH phase SHALL include acceptance tests and generated documentation for all resources and data sources introduced in that phase.
3. EACH phase SHALL produce a tagged release that is installable via `terraform init`.
