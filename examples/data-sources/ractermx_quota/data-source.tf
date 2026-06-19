# Read account quota information to monitor resource usage.
data "ractermx_quota" "current" {}

output "domains_remaining" {
  value = data.ractermx_quota.current.domains_limit - data.ractermx_quota.current.domains_used
}
