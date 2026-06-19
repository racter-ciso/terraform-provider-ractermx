# Create a child organization for team-based domain management.
resource "ractermx_organization" "engineering" {
  name      = "Engineering"
  parent_id = 1
}
