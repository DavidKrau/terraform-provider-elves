# Block an app on the engineering and design configurations
resource "elves_config_rule" "block_tool" {
  rule_id    = elves_rule.blocked_tool.id
  action     = "include"
  config_ids = [2, 3]
}
