package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRuleResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
				resource "elves_rule" "rule" {
					ruletype= "BINARY"
  					policy= "ALLOWLIST"
  					identifier= "this is test identifier"
  					custommessage = "thsi is test message"
  					isdefault= "true"
				  }
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify attributes
					resource.TestCheckResourceAttr("elves_rule.rule", "ruletype", "BINARY"),
					resource.TestCheckResourceAttr("elves_rule.rule", "policy", "ALLOWLIST"),
					resource.TestCheckResourceAttr("elves_rule.rule", "identifier", "this is test identifier"),
					resource.TestCheckResourceAttr("elves_rule.rule", "custommessage", "thsi is test message"),
					resource.TestCheckResourceAttr("elves_rule.rule", "isdefault", "true"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "elves_rule.rule",
				ImportState:       true,
				ImportStateVerify: true,
				//ImportStateVerifyIgnore: []string{"filesha", "mobileconfig"},
			},
			// Update and Read testing
			{
				Config: providerConfig + `
				resource "elves_rule" "rule" {
					ruletype= "TEAMID"
  					policy= "CEL"
  					identifier= "this is new test identifier"
  					custommessage = "thsi is new test message"
  					isdefault= "false"
					celexpression = "aaabbbddd"
				  }
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify attributes
					resource.TestCheckResourceAttr("elves_rule.rule", "ruletype", "TEAMID"),
					resource.TestCheckResourceAttr("elves_rule.rule", "policy", "CEL"),
					resource.TestCheckResourceAttr("elves_rule.rule", "identifier", "this is new test identifier"),
					resource.TestCheckResourceAttr("elves_rule.rule", "custommessage", "thsi is new test message"),
					resource.TestCheckResourceAttr("elves_rule.rule", "isdefault", "false"),
					resource.TestCheckResourceAttr("elves_rule.rule", "celexpression", "aaabbbddd"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
