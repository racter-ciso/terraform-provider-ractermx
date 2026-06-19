# Block all email from a specific sender domain.
resource "ractermx_blocklist_entry" "block_spam_domain" {
  pattern = "*@spam.example.com"
}

# Block a specific sender address.
resource "ractermx_blocklist_entry" "block_sender" {
  pattern = "phishing@malicious.example.com"
}
