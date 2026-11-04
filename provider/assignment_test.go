package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccScimGroupRuleResource(t *testing.T) {
	config := func(action string) string {
		return providerConfig + `
data "elves_scim_group" "group" {
  display_name = "Developers"
}
resource "elves_rule" "rule" {
  ruletype      = "TEAMID"
  policy        = "ALLOWLIST"
  identifier    = "ABCDE12345"
  custommessage = "group rule test"
  isdefault     = false
}
resource "elves_scim_group_rule" "assignment" {
  group_id = data.elves_scim_group.group.id
  rule_id  = elves_rule.rule.id
` + action + `
}
`
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing, action defaults to include
			{
				Config: config(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("elves_scim_group_rule.assignment", "group_id", "data.elves_scim_group.group", "id"),
					resource.TestCheckResourceAttrPair("elves_scim_group_rule.assignment", "rule_id", "elves_rule.rule", "id"),
					resource.TestCheckResourceAttr("elves_scim_group_rule.assignment", "action", "include"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "elves_scim_group_rule.assignment",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: config(`action = "exclude"`),
				Check:  resource.TestCheckResourceAttr("elves_scim_group_rule.assignment", "action", "exclude"),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccClientRuleResource(t *testing.T) {
	config := func(action string) string {
		return providerConfig + `
resource "elves_rule" "rule" {
  ruletype      = "TEAMID"
  policy        = "ALLOWLIST"
  identifier    = "ABCDE12345"
  custommessage = "client rule test"
  isdefault     = false
}
resource "elves_client_rule" "assignment" {
  serial_number = "C02TESTSERIAL"
  rule_id       = elves_rule.rule.id
` + action + `
}
`
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(`action = "exclude"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("elves_client_rule.assignment", "serial_number", "C02TESTSERIAL"),
					resource.TestCheckResourceAttr("elves_client_rule.assignment", "action", "exclude"),
				),
			},
			{
				ResourceName:      "elves_client_rule.assignment",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: config(`action = "include"`),
				Check:  resource.TestCheckResourceAttr("elves_client_rule.assignment", "action", "include"),
			},
		},
	})
}

func TestAccScimUserRuleResource(t *testing.T) {
	config := func(users, action string) string {
		return providerConfig + `
resource "elves_rule" "rule" {
  ruletype      = "TEAMID"
  policy        = "ALLOWLIST"
  identifier    = "ABCDE12345"
  custommessage = "user rule test"
  isdefault     = false
}
resource "elves_scim_user_rule" "assignment" {
  rule_id = elves_rule.rule.id
  users   = [` + users + `]
` + action + `
}
`
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Unknown users are refused before anything is assigned
			{
				Config:      config(`"jane.doe@example.com", "nobody@example.com"`, ""),
				ExpectError: regexp.MustCompile(`unknown SCIM user: nobody@example.com`),
			},
			// Matched by email, or by user name when the user has no email
			{
				Config: config(`"jane.doe@example.com", "John.Roe@Example.com", "max@example.com"`, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("elves_scim_user_rule.assignment", "action", "include"),
					resource.TestCheckResourceAttr("elves_scim_user_rule.assignment", "users.#", "3"),
					resource.TestCheckTypeSetElemAttr("elves_scim_user_rule.assignment", "users.*", "max@example.com"),
				),
			},
			// Import brings in every user that has the rule with that action
			{
				ResourceName:      "elves_scim_user_rule.assignment",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Email case does not matter; drop a user and switch to exclude
			{
				Config: config(`"JANE.DOE@example.com", "John.Roe@Example.com"`, `action = "exclude"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("elves_scim_user_rule.assignment", "action", "exclude"),
					resource.TestCheckResourceAttr("elves_scim_user_rule.assignment", "users.#", "2"),
					resource.TestMatchResourceAttr("elves_scim_user_rule.assignment", "id", regexp.MustCompile(`^[0-9]+/exclude$`)),
				),
			},
		},
	})
}

func TestAccConfigRuleResource(t *testing.T) {
	config := func(ids, action string) string {
		return providerConfig + `
resource "elves_rule" "rule" {
  ruletype      = "TEAMID"
  policy        = "BLOCKLIST"
  identifier    = "ABCDE12345"
  custommessage = "config rule test"
  isdefault     = false
}
resource "elves_config_rule" "assignment" {
  rule_id    = elves_rule.rule.id
  config_ids = [` + ids + `]
` + action + `
}
`
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("2, 3", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("elves_config_rule.assignment", "config_ids.#", "2"),
					resource.TestCheckResourceAttr("elves_config_rule.assignment", "action", "include"),
				),
			},
			{
				ResourceName:      "elves_config_rule.assignment",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: config("1, 3", `action = "exclude"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckTypeSetElemAttr("elves_config_rule.assignment", "config_ids.*", "1"),
					resource.TestCheckTypeSetElemAttr("elves_config_rule.assignment", "config_ids.*", "3"),
					resource.TestCheckResourceAttr("elves_config_rule.assignment", "action", "exclude"),
				),
			},
		},
	})
}

func TestAccConfigurationResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Defaults match the Elves UI
			{
				Config: providerConfig + `
resource "elves_configuration" "config" {
  name        = "tf test"
  client_mode = "MONITOR"
}
data "elves_configuration" "default" {
  name = "default"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("elves_configuration.config", "sync_type", "CLEAN_ALL"),
					resource.TestCheckResourceAttr("elves_configuration.config", "batch_size", "50000"),
					resource.TestCheckResourceAttr("elves_configuration.config", "enable_bundles", "true"),
					resource.TestCheckResourceAttr("elves_configuration.config", "full_sync_interval", "600"),
					resource.TestCheckResourceAttr("elves_configuration.config", "override_file_access_action", "NONE"),
					resource.TestCheckResourceAttr("data.elves_configuration.default", "id", "1"),
				),
			},
			{
				ResourceName:      "elves_configuration.config",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:  "elves_configuration.config",
				ImportState:   true,
				ImportStateId: "1",
				ExpectError:   regexp.MustCompile(`default configuration`),
			},
			{
				Config: providerConfig + `
resource "elves_configuration" "config" {
  name             = "tf test renamed"
  client_mode      = "LOCKDOWN"
  sync_type        = "NORMAL"
  block_usb_mount  = true
  remount_usb_mode = "noexec,rdonly"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("elves_configuration.config", "name", "tf test renamed"),
					resource.TestCheckResourceAttr("elves_configuration.config", "client_mode", "LOCKDOWN"),
					resource.TestCheckResourceAttr("elves_configuration.config", "remount_usb_mode", "noexec,rdonly"),
				),
			},
		},
	})
}
