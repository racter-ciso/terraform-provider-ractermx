# Create an API key with scoped permissions.
resource "ractermx_api_key" "ci" {
  name   = "ci-pipeline"
  scopes = ["email:send", "domains:read"]

  expires_at = "2025-12-31T23:59:59Z"
}

output "api_key_id" {
  value = ractermx_api_key.ci.id
}
