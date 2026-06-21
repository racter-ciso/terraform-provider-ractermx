package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDomain_basic(t *testing.T) {
	domainName := fmt.Sprintf("test-%s.example.com", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and read
			{
				Config: testAccDomainConfig(domainName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ractermx_domain.test", "name", domainName),
					resource.TestCheckResourceAttr("ractermx_domain.test", "catch_all_enabled", "false"),
					resource.TestCheckResourceAttrSet("ractermx_domain.test", "id"),
					resource.TestCheckResourceAttrSet("ractermx_domain.test", "created_at"),
				),
			},
			// Import
			{
				ResourceName:      "ractermx_domain.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				Config: testAccDomainConfigUpdated(domainName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ractermx_domain.test", "catch_all_enabled", "true"),
					resource.TestCheckResourceAttr("ractermx_domain.test", "max_aliases", "500"),
				),
			},
		},
	})
}

func TestAccDomain_withDnsMode(t *testing.T) {
	domainName := fmt.Sprintf("test-%s.example.com", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainConfigWithDnsMode(domainName, "scan_only"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ractermx_domain.test", "dns_mode", "scan_only"),
				),
			},
			{
				Config: testAccDomainConfigWithDnsMode(domainName, "mx_forwarding"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ractermx_domain.test", "dns_mode", "mx_forwarding"),
				),
			},
		},
	})
}

func testAccDomainConfig(name string) string {
	return fmt.Sprintf(`
resource "ractermx_domain" "test" {
  name = %q
}
`, name)
}

func testAccDomainConfigUpdated(name string) string {
	return fmt.Sprintf(`
resource "ractermx_domain" "test" {
  name              = %q
  catch_all_enabled = true
  catch_all_forward_to = "catchall@example.com"
  max_aliases       = 500
}
`, name)
}

func testAccDomainConfigWithDnsMode(name, dnsMode string) string {
	return fmt.Sprintf(`
resource "ractermx_domain" "test" {
  name     = %q
  dns_mode = %q
}
`, name, dnsMode)
}
