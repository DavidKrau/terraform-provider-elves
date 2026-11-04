data "elves_scim_group" "developers" {
  display_name = "Developers"
}

# Allow Docker for everyone in the Developers group
resource "elves_scim_group_rule" "developers_docker" {
  group_id = data.elves_scim_group.developers.id
  rule_id  = elves_rule.docker.id
  action   = "include"
}

# Don't apply a rule from the configuration to this group
resource "elves_scim_group_rule" "developers_no_block" {
  group_id = data.elves_scim_group.developers.id
  rule_id  = elves_rule.blocked_tool.id
  action   = "exclude"
}
