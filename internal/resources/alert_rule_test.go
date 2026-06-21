package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAlertRule_basic(t *testing.T) {
	domainName := fmt.Sprintf("test-%s.example.com", acctest.RandString(8))
	ruleName := fmt.Sprintf("Test Rule %s", acctest.RandString(6))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccAlertRuleConfig(domainName, ruleName, "deliverability_score", "below", "B"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ractermx_alert_rule.test", "name", ruleName),
					resource.TestCheckResourceAttr("ractermx_alert_rule.test", "alert_type", "deliverability_score"),
					resource.TestCheckResourceAttr("ractermx_alert_rule.test", "condition", "below"),
					resource.TestCheckResourceAttr("ractermx_alert_rule.test", "threshold_value", "B"),
					resource.TestCheckResourceAttrSet("ractermx_alert_rule.test", "id"),
				),
			},
			// Import
			{
				ResourceName:      "ractermx_alert_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				Config: testAccAlertRuleConfig(domainName, ruleName+" Updated", "deliverability_score", "below", "C"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ractermx_alert_rule.test", "name", ruleName+" Updated"),
					resource.TestCheckResourceAttr("ractermx_alert_rule.test", "threshold_value", "C"),
				),
			},
		},
	})
}

func testAccAlertRuleConfig(domain, name, alertType, condition, threshold string) string {
	return fmt.Sprintf(`
resource "ractermx_domain" "test" {
  name = %q
}

resource "ractermx_alert_rule" "test" {
  domain_id       = ractermx_domain.test.id
  name            = %q
  alert_type      = %q
  condition       = %q
  threshold_value = %q

  channels = [
    {
      channel_type  = "email"
      email_address = "alerts@example.com"
    }
  ]
}
`, domain, name, alertType, condition, threshold)
}
