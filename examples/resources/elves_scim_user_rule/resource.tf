# Allow Wireshark for a few people only
resource "elves_scim_user_rule" "wireshark_users" {
  rule_id = elves_rule.wireshark.id
  action  = "include"
  users = [
    "jane.doe@example.com",
    "john.roe@example.com",
  ]
}

# Lift a rule these users get from their configuration or a group
resource "elves_scim_user_rule" "no_block_for_admins" {
  rule_id = elves_rule.blocked_tool.id
  action  = "exclude"
  users   = ["it.admin@example.com"]
}
