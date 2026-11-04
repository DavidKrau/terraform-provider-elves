# Import by <rule_id>/<action>. Every user that has the rule with that action
# is imported; remove the ones you don't want Terraform to manage afterwards.
terraform import elves_scim_user_rule.example 68/include
