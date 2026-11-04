package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRuleDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: providerConfig + `data "elves_rule" "test" {id = "5"}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify returned values
					resource.TestCheckResourceAttr("data.elves_rule.test", "ruletype", "BINARY"),
					resource.TestCheckResourceAttr("data.elves_rule.test", "policy", "ALLOWLIST_COMPILER"),
					resource.TestCheckResourceAttr("data.elves_rule.test", "identifier", "wergwer"),
					resource.TestCheckResourceAttr("data.elves_rule.test", "custommessage", "gergewrg"),
					resource.TestCheckResourceAttr("data.elves_rule.test", "isdefault", "true"),
				),
			},
		},
	})
}
