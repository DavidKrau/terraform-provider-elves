package provider

import (
	"strings"

	"github.com/DavidKrau/elves-go-client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// Schema attributes shared by the rule assignment resources.

func assignmentRuleIDAttribute() schema.Int64Attribute {
	return schema.Int64Attribute{
		Required:    true,
		Description: "Required. ID of the rule to assign, for example `elves_rule.example.id`. Changing it replaces the assignment.",
		PlanModifiers: []planmodifier.Int64{
			int64planmodifier.RequiresReplace(),
		},
	}
}

func assignmentActionAttribute() schema.StringAttribute {
	return schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Default:     stringdefault.StaticString("include"),
		Description: "Optional. `include` adds the rule. `exclude` takes it away when an earlier level adds it: rules are applied as default rules, then the configuration, then SCIM groups, then the SCIM user, then the Mac itself. Defaults to `include`.",
		Validators: []validator.String{
			stringvalidator.OneOf("include", "exclude"),
		},
	}
}

func assignmentIDAttribute(format string) schema.StringAttribute {
	return schema.StringAttribute{
		Computed:    true,
		Description: "ID of the assignment, `" + format + "`. Used for import.",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
}

// findAssignedRule returns the assigned rule with ruleID, or nil.
func findAssignedRule(rules []elves.AssignedRule, ruleID int64) *elves.AssignedRule {
	for i := range rules {
		if rules[i].ID == ruleID {
			return &rules[i]
		}
	}
	return nil
}

// isNotFound reports whether a client error is a 404 from Elves.
func isNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "status code: 404")
}
