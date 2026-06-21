package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccWebhook_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccWebhookConfig("https://example.com/webhook", `["email.forwarded"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ractermx_webhook.test", "url", "https://example.com/webhook"),
					resource.TestCheckResourceAttr("ractermx_webhook.test", "events.#", "1"),
					resource.TestCheckResourceAttr("ractermx_webhook.test", "events.0", "email.forwarded"),
					resource.TestCheckResourceAttrSet("ractermx_webhook.test", "secret"),
				),
			},
			// Import
			{
				ResourceName:            "ractermx_webhook.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"secret"},
			},
			// Update
			{
				Config: testAccWebhookConfig("https://example.com/webhook/v2", `["email.forwarded", "email.bounced"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ractermx_webhook.test", "url", "https://example.com/webhook/v2"),
					resource.TestCheckResourceAttr("ractermx_webhook.test", "events.#", "2"),
				),
			},
		},
	})
}

func testAccWebhookConfig(url, events string) string {
	return fmt.Sprintf(`
resource "ractermx_webhook" "test" {
  url    = %q
  events = %s
}
`, url, events)
}
