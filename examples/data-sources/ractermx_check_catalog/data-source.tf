# Read the available security check catalog.
data "ractermx_check_catalog" "all" {}

output "available_checks" {
  value = data.ractermx_check_catalog.all.checks
}
