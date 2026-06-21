package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAlias_basic(t *testing.T) {
	domainName := fmt.Sprintf("test-%s.example.com", acctest.RandString(8))
	localPart := fmt.Sprintf("test-%s", acctest.RandString(6))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccAliasConfig(domainName, localPart, "dest@example.com"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ractermx_alias.test", "local_part", localPart),
					resource.TestCheckResourceAttr("ractermx_alias.test", "forward_to", "dest@example.com"),
					resource.TestCheckResourceAttrSet("ractermx_alias.test", "id"),
				),
			},
			// Import
			{
				ResourceName:      "ractermx_alias.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update forward_to
			{
				Config: testAccAliasConfig(domainName, localPart, "updated@example.com"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ractermx_alias.test", "forward_to", "updated@example.com"),
				),
			},
		},
	})
}

func testAccAliasConfig(domain, localPart, forwardTo string) string {
	return fmt.Sprintf(`
resource "ractermx_domain" "test" {
  name = %q
}

resource "ractermx_alias" "test" {
  domain_id  = ractermx_domain.test.id
  local_part = %q
  forward_to = %q
}
`, domain, localPart, forwardTo)
}
