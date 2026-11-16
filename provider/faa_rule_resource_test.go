package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccFAARuleResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing, with the defaults
			{
				Config: providerConfig + `
resource "elves_faa_rule" "rule" {
  name      = "TerraformTest"
  rule_type = "PathsWithAllowedProcesses"
  paths     = [{ path = "/Users/*/Library/TerraformTest" }]
  processes = [{ team_id = "ABCDE12345", signing_id = "com.example.app" }]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("elves_faa_rule.rule", "id"),
					resource.TestCheckResourceAttr("elves_faa_rule.rule", "name", "TerraformTest"),
					resource.TestCheckResourceAttr("elves_faa_rule.rule", "paths.0.is_prefix", "false"),
					resource.TestCheckResourceAttr("elves_faa_rule.rule", "processes.0.team_id", "ABCDE12345"),
					resource.TestCheckNoResourceAttr("elves_faa_rule.rule", "processes.0.platform_binary"),
					resource.TestCheckResourceAttr("elves_faa_rule.rule", "audit_only", "true"),
					resource.TestCheckResourceAttr("elves_faa_rule.rule", "parent_directory_protection", "enforce"),
					resource.TestCheckResourceAttr("elves_faa_rule.rule", "skip_default_processes", "false"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "elves_faa_rule.rule",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"warnings"},
			},
			// Update and Read testing
			{
				Config: providerConfig + `
resource "elves_faa_rule" "rule" {
  name      = "TerraformTest2"
  rule_type = "ProcessesWithDeniedPaths"
  paths = [
    { path = "/private/tmp/TerraformTest", is_prefix = true },
    { path = "/Users/*/Documents" },
  ]
  processes = [
    { signing_id = "com.apple.curl", platform_binary = true },
    { binary_path = "/usr/local/bin/example" },
  ]
  audit_only                  = false
  allow_read_access           = true
  parent_directory_protection = "disabled"
  block_message               = "Blocked by a test rule"
  event_detail_url            = "https://example.com/faa"
  event_detail_text           = "Details"
  description                 = "Terraform acceptance test"
  skip_default_processes      = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("elves_faa_rule.rule", "name", "TerraformTest2"),
					resource.TestCheckResourceAttr("elves_faa_rule.rule", "rule_type", "ProcessesWithDeniedPaths"),
					resource.TestCheckResourceAttr("elves_faa_rule.rule", "paths.#", "2"),
					resource.TestCheckResourceAttr("elves_faa_rule.rule", "paths.0.is_prefix", "true"),
					resource.TestCheckResourceAttr("elves_faa_rule.rule", "processes.0.platform_binary", "true"),
					resource.TestCheckResourceAttr("elves_faa_rule.rule", "processes.1.binary_path", "/usr/local/bin/example"),
					resource.TestCheckResourceAttr("elves_faa_rule.rule", "audit_only", "false"),
					resource.TestCheckResourceAttr("elves_faa_rule.rule", "parent_directory_protection", "disabled"),
					resource.TestCheckResourceAttr("elves_faa_rule.rule", "event_detail_text", "Details"),
					resource.TestCheckResourceAttr("elves_faa_rule.rule", "skip_default_processes", "true"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccFAASettingsResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Left out, the default processes keep their current value
			{
				Config: providerConfig + `
resource "elves_faa_settings" "this" {
  event_detail_text = "Test details"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("elves_faa_settings.this", "id", "faa_settings"),
					resource.TestCheckResourceAttrSet("elves_faa_settings.this", "version"),
					resource.TestCheckResourceAttr("elves_faa_settings.this", "event_detail_text", "Test details"),
					resource.TestCheckResourceAttrSet("elves_faa_settings.this", "default_processes.#"),
				),
			},
			{
				Config: providerConfig + `
resource "elves_faa_settings" "this" {
  event_detail_text = "Test details"
  default_processes = [
    { team_id = "4AYE5J54KN", signing_id = "com.sentinelone.sentineld-helper" },
    { signing_id = "platform:com.apple.mds", platform_binary = true },
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("elves_faa_settings.this", "default_processes.#", "2"),
					resource.TestCheckResourceAttr("elves_faa_settings.this", "default_processes.0.team_id", "4AYE5J54KN"),
					resource.TestCheckResourceAttr("elves_faa_settings.this", "default_processes.1.platform_binary", "true"),
					resource.TestCheckNoResourceAttr("elves_faa_settings.this", "default_processes.1.team_id"),
				),
			},
			{
				ResourceName:      "elves_faa_settings.this",
				ImportState:       true,
				ImportStateId:     "faa_settings",
				ImportStateVerify: true,
			},
			// An empty list clears them
			{
				Config: providerConfig + `
resource "elves_faa_settings" "this" {
  event_detail_text = "Test details"
  default_processes = []
}
`,
				Check: resource.TestCheckResourceAttr("elves_faa_settings.this", "default_processes.#", "0"),
			},
		},
	})
}
