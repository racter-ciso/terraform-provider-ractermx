data "ractermx_statistics" "last_30_days" {
  date_from = timeadd(timestamp(), "-720h") # 30 days ago
  date_to   = timestamp()
}

output "total_emails_received" {
  value = data.ractermx_statistics.last_30_days.total_received
}

output "total_emails_forwarded" {
  value = data.ractermx_statistics.last_30_days.total_forwarded
}

output "total_emails_bounced" {
  value = data.ractermx_statistics.last_30_days.total_bounced
}
