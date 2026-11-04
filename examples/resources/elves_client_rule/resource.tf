# Allow a tool on one Mac only
resource "elves_client_rule" "build_mac_tool" {
  serial_number = "C02XK0AAJG5J"
  rule_id       = elves_rule.build_tool.id
  action        = "include"
}

# Lift a rule that this Mac gets from its configuration or a group
resource "elves_client_rule" "build_mac_no_block" {
  serial_number = "C02XK0AAJG5J"
  rule_id       = elves_rule.blocked_tool.id
  action        = "exclude"
}
