# Create SMTP credentials for sending email through a domain.
resource "ractermx_smtp_credential" "example" {
  domain_id   = ractermx_domain.example.id
  daily_limit = 5000
}

resource "ractermx_domain" "example" {
  name = "example.com"
}

output "smtp_username" {
  value = ractermx_smtp_credential.example.username
}

output "smtp_host" {
  value = ractermx_smtp_credential.example.smtp_config.host
}
