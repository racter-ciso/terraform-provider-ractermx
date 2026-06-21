package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccZoneRecord_basic(t *testing.T) {
	domainName := fmt.Sprintf("test-%s.example.com", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create A record
			{
				Config: testAccZoneRecordConfig(domainName, "www", "A", "1.2.3.4", 3600),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ractermx_zone_record.test", "name", "www"),
					resource.TestCheckResourceAttr("ractermx_zone_record.test", "type", "A"),
					resource.TestCheckResourceAttr("ractermx_zone_record.test", "content", "1.2.3.4"),
					resource.TestCheckResourceAttr("ractermx_zone_record.test", "ttl", "3600"),
				),
			},
			// Import
			{
				ResourceName:      "ractermx_zone_record.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update TTL
			{
				Config: testAccZoneRecordConfig(domainName, "www", "A", "1.2.3.4", 7200),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ractermx_zone_record.test", "ttl", "7200"),
				),
			},
		},
	})
}

func testAccZoneRecordConfig(domain, name, recordType, content string, ttl int) string {
	return fmt.Sprintf(`
resource "ractermx_domain" "test" {
  name     = %q
  dns_mode = "dns_hosted"
}

resource "ractermx_zone_record" "test" {
  domain_id = ractermx_domain.test.id
  name      = %q
  type      = %q
  content   = %q
  ttl       = %d
}
`, domain, name, recordType, content, ttl)
}
