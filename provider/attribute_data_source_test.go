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
					resource.TestCheckResourceAttr("elves_rule.rule", "ruletype", "BINARY"),
					resource.TestCheckResourceAttr("elves_rule.rule", "policy", "ALLOWLIST_COMPILER"),
					resource.TestCheckResourceAttr("elves_rule.rule", "identifier", "wergwer"),
					resource.TestCheckResourceAttr("elves_rule.rule", "custommessage", "gergewrg"),
					resource.TestCheckResourceAttr("elves_rule.rule", "isdefault", "true"),
				),
			},
		},
	})
}
