# Manage a RacterMX domain with email forwarding enabled.
resource "ractermx_domain" "example" {
  name              = "example.com"
  catch_all_enabled = true
  catch_all_forward_to = "admin@example.com"
  max_aliases       = 100
}
