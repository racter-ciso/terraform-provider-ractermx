# Implementation Plan: terraform-provider-ractermx

## Overview

This plan implements a Terraform provider for the RacterMX REST API v2 in Go using the HashiCorp Terraform Plugin Framework. The implementation follows a 4-phase delivery approach: Phase 1 covers the provider scaffold, API client, Domain resource, Alias resource, and their data sources; Phase 2 adds Zone Record, Webhook, and Blocklist Entry resources; Phase 3 adds SMTP Credential, API Key, Retention Policy, and Organization resources; Phase 4 adds Alert Rule, Domain Tag, Tag Assignment, Notification Preference, Check Override resources, and remaining data sources.

Each task builds incrementally on previous tasks, and all code is wired into the provider registration before moving to the next phase.

## Tasks

### Phase 1: Provider Scaffold, API Client, Domain, Alias, and Data Sources

- [x] 1. Set up project scaffold, Go module, and build system
  - [x] 1.1 Create `go.mod` with module name `terraform-provider-ractermx` and dependencies: `terraform-plugin-framework`, `terraform-plugin-go`, `terraform-plugin-testing`, `terraform-plugin-log`, `pgregory.net/rapid`
    - Run `go mod tidy` to resolve all dependency versions
    - _Requirements: 25.1, 25.2_
  - [x] 1.2 Create `main.go` entry point that registers the provider with `providerserver.Serve`
    - Include `go:generate` directive for `tfplugindocs generate`
    - _Requirements: 25.6, 25.5_
  - [x] 1.3 Create `GNUmakefile` with targets: `build`, `install`, `test`, `testacc`, `generate`, `lint`, `fmt`
    - _Requirements: 25.3_
  - [x] 1.4 Create `.goreleaser.yml` for cross-platform builds (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64)
    - _Requirements: 25.4_
  - [x] 1.5 Create directory structure: `internal/provider/`, `internal/client/`, `internal/resources/`, `internal/datasources/`, `examples/`, `templates/`, `docs/`
    - _Requirements: 25.1_

- [x] 2. Implement the API client package (`internal/client/`)
  - [x] 2.1 Create `client.go` with `Client` struct (BaseURL, APIKey, HTTPClient, UserAgent) and constructor `NewClient(apiKey, baseURL, version string) *Client`
    - Set default HTTP timeout to 30 seconds
    - Append `/api/v2` to the base URL
    - Set `User-Agent` header to `terraform-provider-ractermx/<version>`
    - _Requirements: 1.5, 1.6, 2.5, 2.6_
  - [x] 2.2 Implement HTTP methods: `Get`, `Post`, `Patch`, `Put`, `Delete`, `DeleteWithBody` on the `Client` struct
    - Each method adds `Authorization: Bearer {api_key}` header
    - Each method adds `Content-Type: application/json` for request bodies
    - _Requirements: 1.5, 2.6_
  - [x] 2.3 Create `errors.go` with API error parsing: parse 422 `errors` object into field-level diagnostics, handle 401/404/409/429/5xx status codes
    - 404 during Read returns nil (resource removed from state)
    - 401 returns descriptive "Invalid or expired API credentials" error
    - 422 parses `errors` map into per-field Terraform diagnostics
    - _Requirements: 1.7, 2.1, 2.3_
  - [x] 2.4 Create `retry.go` with exponential backoff: retry 429 up to 3 times (1s/2s/4s), retry 5xx up to 2 times (1s/2s)
    - _Requirements: 2.2, 2.4_
  - [x] 2.5 Create `pagination.go` with `ListAll` method that iterates paginated responses using `meta.current_page` / `meta.last_page`, fetching with `?page=N&per_page=100`
    - Handle non-paginated list endpoints (webhooks, blocklist, tags) by returning `data` array directly
    - _Requirements: 2.7_

  - [x] 2.6 Write property test for HTTP request construction (Property 2)
    - **Property 2: HTTP request construction**
    - For any non-empty API key, valid base URL, and version string, verify every request includes correct `Authorization`, `User-Agent`, and URL prefix
    - Test file: `internal/client/request_test.go`
    - **Validates: Requirements 1.5, 1.6, 2.6**
  - [x] 2.7 Write property test for API error response parsing (Property 3)
    - **Property 3: API error response parsing**
    - For any valid JSON `errors` object with field-name-to-string-array map, verify one diagnostic per field with correct detail
    - Test file: `internal/client/errors_test.go`
    - **Validates: Requirements 2.1**
  - [x] 2.8 Write property test for retry behavior (Property 4)
    - **Property 4: Retry behavior for transient errors**
    - For 429 status: succeed when N ≤ 3 consecutive errors, fail when N > 3. For 5xx: succeed when N ≤ 2, fail when N > 2. Verify exponential backoff ordering.
    - Test file: `internal/client/retry_test.go`
    - **Validates: Requirements 2.2, 2.4**
  - [x] 2.9 Write property test for pagination completeness (Property 5)
    - **Property 5: Pagination completeness**
    - For any total T and page size P, verify ListAll returns exactly T items by fetching ⌈T/P⌉ pages in order
    - Test file: `internal/client/pagination_test.go`
    - **Validates: Requirements 2.7**

- [x] 3. Implement the provider (`internal/provider/provider.go`)
  - [x] 3.1 Create `provider.go` implementing `provider.Provider` interface with `Metadata`, `Schema`, `Configure`, `Resources`, `DataSources` methods
    - Schema: `api_key` (required, sensitive), `base_url` (optional, default `https://ractermx.com`)
    - Configure: resolve `api_key` from HCL or `RACTERMX_API_KEY` env var; return diagnostic error if both absent
    - Construct `client.Client` and store in provider data
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.8_
  - [ ]* 3.2 Write property test for provider configuration resolution (Property 1)
    - **Property 1: Provider configuration resolution**
    - For any combination of (api_key in HCL present/absent) × (env var present/absent) × (base_url present/absent), verify correct resolution, fallback, and error behavior
    - Test file: `internal/provider/provider_config_test.go`
    - **Validates: Requirements 1.2, 1.3, 1.4**

- [x] 4. Checkpoint - Verify scaffold builds and tests pass
  - Ensure `go build ./...` succeeds, `go test ./internal/client/... ./internal/provider/...` passes, ask the user if questions arise.

- [x] 5. Implement Domain resource (`internal/resources/domain.go`)
  - [x] 5.1 Create `DomainResource` struct implementing `resource.Resource` with `Metadata`, `Schema`, `Configure`, `Create`, `Read`, `Update`, `Delete`, `ImportState` methods
    - Define `DomainResourceModel` with all attributes: `id`, `name`, `organization_id`, `is_forwarding`, `is_monitored`, `is_hosted`, `dns_mode`, `catch_all_enabled`, `catch_all_forward_to`, `max_aliases`, and computed: `is_active`, `is_verified`, `verification_token`, `mx_verified`, `spf_verified`, `dkim_verified`, `dmarc_verified`, `last_verified_at`, `created_at`, `updated_at`
    - `name` uses `RequiresReplace()` plan modifier
    - Create maps to `POST /domains`, Read to `GET /domains/{id}`, Update to `PATCH /domains/{id}`, Delete to `DELETE /domains/{id}`
    - Import uses numeric domain ID
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.7_
  - [x] 5.2 Register `DomainResource` in the provider's `Resources()` method
    - _Requirements: 3.1_

- [x] 6. Implement Domain Verification resource (`internal/resources/domain_verification.go`)
  - [x] 6.1 Create `DomainVerificationResource` implementing `resource.Resource`
    - Create triggers `POST /domains/{id}/verify-dns`
    - Read calls `GET /domains/{id}` to refresh verification status
    - Delete is a no-op
    - Expose computed: `mx_verified`, `spf_verified`, `dkim_verified`, `dmarc_verified`, `is_verified`
    - Support re-verification via taint
    - _Requirements: 4.1, 4.2, 4.3, 4.4_
  - [x] 6.2 Register `DomainVerificationResource` in the provider's `Resources()` method
    - _Requirements: 4.1_

- [x] 7. Implement Alias resource (`internal/resources/alias.go`)
  - [x] 7.1 Create `AliasResource` implementing `resource.Resource` with full CRUD and import
    - Define `AliasResourceModel`: `id`, `domain_id`, `local_part`, `forward_to`, `is_catchall`, `description`, and computed: `is_active`, `is_wildcard`, `created_at`, `updated_at`
    - When `is_catchall` is true, set `local_part` to `*`
    - `local_part` and `domain_id` use `RequiresReplace()` plan modifier
    - Create maps to `POST /domains/{domainId}/aliases`, Read to `GET /aliases/{id}`, Update to `PATCH /aliases/{id}`, Delete to `DELETE /aliases/{id}`
    - Handle 409 Conflict with clear alias address in error message
    - Import uses numeric alias ID
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7, 5.8_
  - [x] 7.2 Register `AliasResource` in the provider's `Resources()` method
    - _Requirements: 5.1_

- [x] 8. Implement Phase 1 data sources
  - [x] 8.1 Create `DomainDnsRecordsDataSource` (`internal/datasources/domain_dns_records.go`)
    - Read from `GET /domains/{id}/dns-records`
    - Expose `mx`, `spf`, `dkim`, `dmarc` nested objects with `type`, `name`, `value`, `ttl`
    - _Requirements: 18.1, 18.2, 18.3_
  - [x] 8.2 Create `DomainStatisticsDataSource` (`internal/datasources/domain_statistics.go`)
    - Read from `GET /domains/{id}/statistics`
    - Accept optional `date_from` and `date_to` attributes
    - Expose `total_received`, `total_forwarded`, `total_bounced`, `total_deferred`, `total_rejected`
    - _Requirements: 19.1, 19.2, 19.3, 19.4_
  - [x] 8.3 Create `DomainHealthDataSource` (`internal/datasources/domain_health.go`)
    - Read from `GET /domains/{id}/health`
    - Expose `overall_status`, `domain_verified`, and `checks` nested object with `mx`, `spf`, `dkim`, `dmarc` sub-objects
    - _Requirements: 20.1, 20.2, 20.3_
  - [x] 8.4 Register all Phase 1 data sources in the provider's `DataSources()` method
    - _Requirements: 18.1, 19.1, 20.1_

- [x] 9. Checkpoint - Phase 1 build and unit tests
  - Ensure `go build ./...` succeeds, all unit and property tests pass with `go test ./internal/... -v`, ask the user if questions arise.

- [ ]* 10. Write Phase 1 acceptance tests
  - [ ]* 10.1 Create test helpers: `testAccPreCheck`, `testAccProtoV6ProviderFactories`, randomized name generators
    - Skip tests when `RACTERMX_API_KEY` is not set
    - _Requirements: 26.3, 26.4, 26.6_
  - [ ]* 10.2 Write acceptance tests for `ractermx_domain`: basic CRUD lifecycle, import, force-replace on name change, `CheckDestroy`
    - _Requirements: 26.1, 26.5_
  - [ ]* 10.3 Write acceptance tests for `ractermx_domain_verification`: create + read verification status
    - _Requirements: 26.1_
  - [ ]* 10.4 Write acceptance tests for `ractermx_alias`: CRUD lifecycle, import, catchall behavior, 409 conflict, force-replace on local_part change, `CheckDestroy`
    - _Requirements: 26.1, 26.5_
  - [ ]* 10.5 Write acceptance tests for Phase 1 data sources: `ractermx_domain_dns_records`, `ractermx_domain_statistics`, `ractermx_domain_health`
    - _Requirements: 26.2_

- [x] 11. Create Phase 1 example HCL files and generate documentation
  - Create `examples/provider/`, `examples/resources/ractermx_domain/`, `examples/resources/ractermx_alias/`, `examples/resources/ractermx_domain_verification/`, `examples/data-sources/ractermx_domain_dns_records/`, `examples/data-sources/ractermx_domain_statistics/`, `examples/data-sources/ractermx_domain_health/` with `main.tf` example files
  - _Requirements: 25.5, 27.2_

### Phase 2: Zone Record, Webhook, Blocklist Entry

- [x] 12. Implement Zone Record resource (`internal/resources/zone_record.go`)
  - [x] 12.1 Create composite ID helpers: `FormatZoneRecordID(domainID, name, recordType, content)` and `ParseZoneRecordID(id)` in a shared `internal/resources/composite_id.go` file
    - Format: `{domain_id}/{name}/{type}/{content}` with `/` separator
    - _Requirements: 6.6_
  - [x] 12.2 Create `ZoneRecordResource` implementing `resource.Resource` with full CRUD and import
    - Define `ZoneRecordResourceModel`: `domain_id`, `name`, `type`, `content`, `ttl`, `priority`, `weight`, `port`
    - `domain_id` uses `RequiresReplace()` plan modifier
    - Create maps to `POST /domains/{domainId}/zone-records`
    - Read lists all records via `GET /domains/{domainId}/zone-records` and matches by name+type+content
    - Update sends old/new pattern to `PATCH /domains/{domainId}/zone-records`
    - Delete sends JSON body to `DELETE /domains/{domainId}/zone-records`
    - Import parses composite key `{domain_id}/{name}/{type}/{content}`
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 6.7_
  - [x] 12.3 Register `ZoneRecordResource` in the provider's `Resources()` method
    - _Requirements: 6.1_
  - [x] 12.4 Write property test for composite ID round-trip (Property 6 — zone record)
    - **Property 6: Composite ID round-trip (zone record)**
    - For any valid (domain_id, name, type, content), verify format→parse round-trip yields original components
    - Test file: `internal/resources/composite_id_test.go`
    - **Validates: Requirements 6.6**

- [x] 13. Implement Webhook resource (`internal/resources/webhook.go`)
  - [x] 13.1 Create `WebhookResource` implementing `resource.Resource` with full CRUD and import
    - Define `WebhookResourceModel`: `id`, `url`, `events`, `custom_headers`, `timeout_seconds`, `batch_enabled`, `enabled`, and computed sensitive: `secret`, `created_at`
    - `secret` uses `Sensitive: true`, `Computed: true`, `UseStateForUnknown()` — stored on create, preserved on read
    - Create maps to `POST /webhooks`, Read lists via `GET /webhooks` and matches by ID, Update to `PUT /webhooks/{id}`, Delete to `DELETE /webhooks/{id}`
    - Import uses numeric webhook ID; `secret` will be empty after import
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5, 7.6, 7.7_
  - [x] 13.2 Register `WebhookResource` in the provider's `Resources()` method
    - _Requirements: 7.1_

- [x] 14. Implement Blocklist Entry resource (`internal/resources/blocklist_entry.go`)
  - [x] 14.1 Create `BlocklistEntryResource` implementing `resource.Resource` with Create, Read, Delete, and Import (no Update)
    - Define `BlocklistEntryResourceModel`: `id`, `pattern`, `created_at`
    - `pattern` uses `RequiresReplace()` plan modifier
    - Create maps to `POST /blocklist`, Read lists via `GET /blocklist` and matches by ID, Delete to `DELETE /blocklist/{id}`
    - Handle 409 Conflict for duplicate patterns
    - Import uses numeric blocklist entry ID
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 8.6_
  - [x] 14.2 Register `BlocklistEntryResource` in the provider's `Resources()` method
    - _Requirements: 8.1_

- [x] 15. Checkpoint - Phase 2 build and unit tests
  - Ensure `go build ./...` succeeds, all unit and property tests pass, ask the user if questions arise.

- [ ]* 16. Write Phase 2 acceptance tests
  - [ ]* 16.1 Write acceptance tests for `ractermx_zone_record`: CRUD lifecycle with old/new update pattern, import with composite key, `CheckDestroy`
    - _Requirements: 26.1, 26.5_
  - [ ]* 16.2 Write acceptance tests for `ractermx_webhook`: CRUD lifecycle, import (verify secret is empty on import), `CheckDestroy`
    - _Requirements: 26.1, 26.5_
  - [ ]* 16.3 Write acceptance tests for `ractermx_blocklist_entry`: Create-Read-Delete lifecycle, import, 409 conflict, force-replace on pattern change, `CheckDestroy`
    - _Requirements: 26.1, 26.5_

- [x] 17. Create Phase 2 example HCL files
  - Create `examples/resources/ractermx_zone_record/`, `examples/resources/ractermx_webhook/`, `examples/resources/ractermx_blocklist_entry/` with `main.tf` and `import.sh` example files
  - _Requirements: 25.5, 27.2_

### Phase 3: SMTP Credential, API Key, Retention Policy, Organization

- [x] 18. Implement SMTP Credential resource (`internal/resources/smtp_credential.go`)
  - [x] 18.1 Create `SmtpCredentialResource` implementing `resource.Resource` with Create, Read, Delete, and Import (no Update)
    - Define `SmtpCredentialResourceModel`: `id`, `domain_id`, `daily_limit`, `anonymous_reply_enabled`, `proxy_domain_id`, and computed: `username`, `password` (sensitive), `smtp_config` (nested object with host, port, encryption)
    - `password` uses `Sensitive: true`, `Computed: true`, `UseStateForUnknown()` — stored on create, preserved on read
    - All configurable attributes use `RequiresReplace()` since there is no update endpoint
    - Create maps to `POST /domains/{domainId}/smtp-credentials`, Read lists via `GET /domains/{domainId}/smtp-credentials` and matches by ID, Delete to `DELETE /smtp-credentials/{id}`
    - Import uses numeric credential ID; `password` will be empty after import
    - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5, 9.6, 9.7, 9.8_
  - [x] 18.2 Register `SmtpCredentialResource` in the provider's `Resources()` method
    - _Requirements: 9.1_

- [x] 19. Implement API Key resource (`internal/resources/api_key.go`)
  - [x] 19.1 Create `ApiKeyResource` implementing `resource.Resource` with Create, Read, Delete, and Import (no Update)
    - Define `ApiKeyResourceModel`: `id`, `name`, `scopes`, `expires_at`, `allowed_ips`, and computed: `api_key` (sensitive), `last_used_at`, `created_at`
    - `api_key` uses `Sensitive: true`, `Computed: true`, `UseStateForUnknown()` — stored on create, preserved on read
    - All configurable attributes use `RequiresReplace()` since there is no update endpoint
    - Create maps to `POST /api-keys`, Read lists via `GET /api-keys` and matches by ID, Delete to `DELETE /api-keys/{id}`
    - Import uses numeric API key ID; `api_key` will be empty after import
    - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5, 10.6, 10.7, 10.8_
  - [x] 19.2 Register `ApiKeyResource` in the provider's `Resources()` method
    - _Requirements: 10.1_

- [x] 20. Implement Retention Policy resource (`internal/resources/retention_policy.go`)
  - [x] 20.1 Create `RetentionPolicyResource` implementing `resource.Resource` with singleton pattern
    - Define `RetentionPolicyResourceModel`: `metadata_retention_days`, `event_specific_retention` (map of string to int), and computed: `updated_at`
    - Create and Update both map to `PUT /retention-policy` (upsert)
    - Read maps to `GET /retention-policy`
    - Delete is a no-op that logs a warning (policy remains on server)
    - Import uses fixed ID `"default"`
    - _Requirements: 11.1, 11.2, 11.3, 11.4, 11.5, 11.6_
  - [x] 20.2 Register `RetentionPolicyResource` in the provider's `Resources()` method
    - _Requirements: 11.1_

- [x] 21. Implement Organization resource (`internal/resources/organization.go`)
  - [x] 21.1 Create `OrganizationResource` implementing `resource.Resource` with full CRUD and import
    - Define `OrganizationResourceModel`: `id`, `name`, `parent_id`, and computed: `users_count`, `domains_count`, `total_domains_count`
    - Create maps to `POST /organizations`, Read traverses `GET /organizations` tree to find by ID, Update to `PATCH /organizations/{id}`, Delete to `DELETE /organizations/{id}`
    - Handle delete precondition errors (domains, children, members must be removed first)
    - Handle primary organization delete protection
    - Import uses numeric organization ID
    - _Requirements: 12.1, 12.2, 12.3, 12.4, 12.5, 12.6_
  - [x] 21.2 Register `OrganizationResource` in the provider's `Resources()` method
    - _Requirements: 12.1_

- [x] 22. Implement Quota data source (`internal/datasources/quota.go`)
  - [x] 22.1 Create `QuotaDataSource` reading from `GET /quota`
    - Expose quota attributes returned by the API
    - _Requirements: 24.1, 24.2_
  - [x] 22.2 Register `QuotaDataSource` in the provider's `DataSources()` method
    - _Requirements: 24.1_

- [x] 23. Checkpoint - Phase 3 build and unit tests
  - Ensure `go build ./...` succeeds, all unit and property tests pass, ask the user if questions arise.

- [ ]* 24. Write Phase 3 acceptance tests
  - [ ] 24.1 Write acceptance tests for `ractermx_smtp_credential`: Create-Read-Delete lifecycle, import (verify password empty), force-replace on attribute change, `CheckDestroy`
    - _Requirements: 26.1, 26.5_
  - [ ] 24.2 Write acceptance tests for `ractermx_api_key`: Create-Read-Delete lifecycle, import (verify api_key empty), force-replace on any change, `CheckDestroy`
    - _Requirements: 26.1, 26.5_
  - [ ] 24.3 Write acceptance tests for `ractermx_retention_policy`: Read-Update lifecycle, import with `"default"` ID, verify no-op delete
    - _Requirements: 26.1_
  - [ ]* 24.4 Write acceptance tests for `ractermx_organization`: CRUD lifecycle, import, delete precondition errors, `CheckDestroy`
    - _Requirements: 26.1, 26.5_
  - [ ]* 24.5 Write acceptance tests for `ractermx_quota` data source
    - _Requirements: 26.2_

- [x] 25. Create Phase 3 example HCL files
  - Create `examples/resources/ractermx_smtp_credential/`, `examples/resources/ractermx_api_key/`, `examples/resources/ractermx_retention_policy/`, `examples/resources/ractermx_organization/`, `examples/data-sources/ractermx_quota/` with `main.tf` example files
  - _Requirements: 25.5, 27.2_

### Phase 4: Alert Rule, Domain Tag, Tag Assignment, Notification Preference, Check Override, and Remaining Data Sources

- [x] 26. Implement Domain Tag resource (`internal/resources/domain_tag.go`)
  - [x] 26.1 Create `DomainTagResource` implementing `resource.Resource` with full CRUD and import
    - Define `DomainTagResourceModel`: `id`, `name`, `color`, and computed: `domains_count`
    - `color` defaults to `#3b82f6`; validate hex format `^#[0-9a-fA-F]{6}$`
    - Create maps to `POST /tags`, Read lists via `GET /tags` and matches by ID, Update to `PATCH /tags/{id}`, Delete to `DELETE /tags/{id}`
    - Handle duplicate name error
    - Import uses numeric tag ID
    - _Requirements: 13.1, 13.2, 13.3, 13.4, 13.5, 13.6_
  - [x] 26.2 Register `DomainTagResource` in the provider's `Resources()` method
    - _Requirements: 13.1_

- [x] 27. Implement Domain Tag Assignment resource (`internal/resources/domain_tag_assignment.go`)
  - [x] 27.1 Add composite ID helpers for tag assignment: `FormatTagAssignmentID(domainID, tagID)` and `ParseTagAssignmentID(id)` in `internal/resources/composite_id.go`
    - Format: `{domain_id}/{tag_id}` with `/` separator
    - _Requirements: 14.3_
  - [x] 27.2 Create `DomainTagAssignmentResource` implementing `resource.Resource` with Create, Read, Delete, and Import (no Update)
    - Define `DomainTagAssignmentResourceModel`: `domain_id`, `tag_id`
    - Both attributes use `RequiresReplace()` plan modifier
    - Create maps to `POST /domains/{id}/tags` with `{ "tag_ids": [tagId] }`
    - Read calls `GET /domains/{id}` and checks if tag is in domain's `tags` array
    - Delete maps to `DELETE /domains/{id}/tags/{tagId}`
    - Import parses composite key `{domain_id}/{tag_id}`
    - _Requirements: 14.1, 14.2, 14.3, 14.4_
  - [x] 27.3 Register `DomainTagAssignmentResource` in the provider's `Resources()` method
    - _Requirements: 14.1_
  - [ ]* 27.4 Write property test for composite ID round-trip (Property 6 — tag assignment)
    - **Property 6: Composite ID round-trip (tag assignment)**
    - For any valid (domain_id, tag_id), verify format→parse round-trip yields original components
    - Test file: `internal/resources/composite_id_test.go` (extend existing file)
    - **Validates: Requirements 14.3**

- [x] 28. Implement Alert Rule resource (`internal/resources/alert_rule.go`)
  - [x] 28.1 Create alert rule cross-field validation function in `internal/resources/alert_rule_validation.go`
    - Validate `blacklist_change` requires `any_change` condition and null threshold
    - Validate `deliverability_score`/`security_posture` requires non-`any_change` condition and valid grade (A-F)
    - Validate `dmarc_compliance` requires non-`any_change` condition and integer 0-100 threshold
    - Reject all other combinations with descriptive error
    - _Requirements: 15.4, 15.5, 15.6_
  - [x] 28.2 Create `AlertRuleResource` implementing `resource.Resource` with full CRUD and import
    - Define `AlertRuleResourceModel`: `id`, `domain_id`, `name`, `alert_type`, `condition`, `threshold_value`, `cooldown_minutes`, `enabled`, `channels` (list nested attribute), and computed: `created_at`
    - `channels` uses `ListNestedAttribute` with `channel_type`, `webhook_endpoint_id`, `email_address`; validated 1-3 items
    - Apply cross-field validation during Create and Update
    - Create maps to `POST /alert-rules`, Read to `GET /alert-rules/{id}`, Update to `PATCH /alert-rules/{id}`, Delete to `DELETE /alert-rules/{id}`
    - Import uses numeric alert rule ID
    - _Requirements: 15.1, 15.2, 15.3, 15.4, 15.5, 15.6, 15.7, 15.8, 15.9_
  - [x] 28.3 Register `AlertRuleResource` in the provider's `Resources()` method
    - _Requirements: 15.1_
  - [ ]* 28.4 Write property test for alert rule cross-field validation (Property 7)
    - **Property 7: Alert rule cross-field validation**
    - For any combination of (alert_type, condition, threshold_value), verify correct accept/reject behavior per the validation rules
    - Test file: `internal/resources/alert_rule_validation_test.go`
    - **Validates: Requirements 15.4, 15.5, 15.6**

- [x] 29. Implement Notification Preference resource (`internal/resources/notification_preference.go`)
  - [x] 29.1 Create `NotificationPreferenceResource` implementing `resource.Resource` with upsert semantics
    - Define `NotificationPreferenceResourceModel`: `domain_id`, `muted`, `min_priority`
    - Create and Update both map to `POST /domains/{id}/notification-preferences` (upsert)
    - Read maps to `GET /domains/{id}/notification-preferences`
    - Delete resets to defaults (`muted=false`, `min_priority=null`) rather than deleting
    - Import uses domain numeric ID
    - _Requirements: 16.1, 16.2, 16.3, 16.4, 16.5_
  - [x] 29.2 Register `NotificationPreferenceResource` in the provider's `Resources()` method
    - _Requirements: 16.1_

- [x] 30. Implement Check Override resource (`internal/resources/check_override.go`)
  - [x] 30.1 Add composite ID helpers for check override: `FormatCheckOverrideID(domainID, checkID)` and `ParseCheckOverrideID(id)` in `internal/resources/composite_id.go`
    - Format: `{domain_id}/{check_id}` with `/` separator
    - _Requirements: 17.5_
  - [x] 30.2 Create `CheckOverrideResource` implementing `resource.Resource` with upsert semantics
    - Define `CheckOverrideResourceModel`: `domain_id`, `check_id`, `enabled`, `severity_override`
    - Both `domain_id` and `check_id` use `RequiresReplace()` plan modifier
    - Create and Update both map to `PUT /domains/{id}/check-overrides/{checkId}` (upsert)
    - Read maps to `GET /check-catalog` and matches by domain+check
    - Delete sends null values to reset override to catalog defaults
    - Import parses composite key `{domain_id}/{check_id}`
    - _Requirements: 17.1, 17.2, 17.3, 17.4, 17.5, 17.6_
  - [x] 30.3 Register `CheckOverrideResource` in the provider's `Resources()` method
    - _Requirements: 17.1_
  - [ ]* 30.4 Write property test for composite ID round-trip (Property 6 — check override)
    - **Property 6: Composite ID round-trip (check override)**
    - For any valid (domain_id, check_id), verify format→parse round-trip yields original components
    - Test file: `internal/resources/composite_id_test.go` (extend existing file)
    - **Validates: Requirements 17.5**

- [x] 31. Implement Phase 4 data sources
  - [x] 31.1 Create `SecurityScoreDataSource` (`internal/datasources/security_score.go`)
    - Read from `GET /domains/{id}/security/score`
    - Expose `overall_score`, `grade`, and pillar breakdown attributes
    - _Requirements: 21.1, 21.2, 21.3_
  - [x] 31.2 Create `SecurityChecksDataSource` (`internal/datasources/security_checks.go`)
    - Read from `GET /domains/{id}/security`
    - Expose list of findings grouped by pillar with check details, status, severity, and fix availability
    - _Requirements: 22.1, 22.2, 22.3_
  - [x] 31.3 Create `CheckCatalogDataSource` (`internal/datasources/check_catalog.go`)
    - Read from `GET /check-catalog`
    - Expose list of checks grouped by pillar with `check_id`, `name`, `description`, `default_severity`, `version`
    - _Requirements: 23.1, 23.2_
  - [x] 31.4 Register all Phase 4 data sources in the provider's `DataSources()` method
    - _Requirements: 21.1, 22.1, 23.1_

- [x] 32. Checkpoint - Phase 4 build and unit tests
  - Ensure `go build ./...` succeeds, all unit and property tests pass, ask the user if questions arise.

- [ ]* 33. Write Phase 4 acceptance tests
  - [ ]* 33.1 Write acceptance tests for `ractermx_domain_tag`: CRUD lifecycle, import, duplicate name error, `CheckDestroy`
    - _Requirements: 26.1, 26.5_
  - [ ]* 33.2 Write acceptance tests for `ractermx_domain_tag_assignment`: Create-Delete lifecycle, import with composite key, `CheckDestroy`
    - _Requirements: 26.1, 26.5_
  - [ ]* 33.3 Write acceptance tests for `ractermx_alert_rule`: CRUD lifecycle, import, cross-field validation errors, `CheckDestroy`
    - _Requirements: 26.1, 26.5_
  - [ ]* 33.4 Write acceptance tests for `ractermx_domain_notification_preference`: Create-Read-Delete lifecycle, import, verify reset-to-defaults on delete
    - _Requirements: 26.1_
  - [ ]* 33.5 Write acceptance tests for `ractermx_check_override`: Create-Read-Delete lifecycle, import with composite key, verify reset-to-defaults on delete
    - _Requirements: 26.1_
  - [ ]* 33.6 Write acceptance tests for Phase 4 data sources: `ractermx_security_score`, `ractermx_security_checks`, `ractermx_check_catalog`
    - _Requirements: 26.2_

- [x] 34. Create Phase 4 example HCL files
  - Create `examples/resources/` and `examples/data-sources/` directories for all Phase 4 resources and data sources with `main.tf` example files
  - _Requirements: 25.5, 27.2_

- [x] 35. Final checkpoint - Full build, all tests, documentation generation
  - Ensure `go build ./...` succeeds, all unit and property tests pass, run `make generate` to produce documentation, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation at each phase boundary
- Property tests validate universal correctness properties from the design document (Properties 1-7)
- Acceptance tests verify full CRUD lifecycles against the real RacterMX API
- The implementation language is Go, using the HashiCorp Terraform Plugin Framework
- All composite ID helpers are centralized in `internal/resources/composite_id.go` for consistency
