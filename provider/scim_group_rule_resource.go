package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/DavidKrau/elves-go-client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &scimGroupRuleResource{}
	_ resource.ResourceWithConfigure   = &scimGroupRuleResource{}
	_ resource.ResourceWithImportState = &scimGroupRuleResource{}
)

// scimGroupRuleResourceModel maps the resource schema data.
type scimGroupRuleResourceModel struct {
	GroupID types.Int64  `tfsdk:"group_id"`
	RuleID  types.Int64  `tfsdk:"rule_id"`
	Action  types.String `tfsdk:"action"`
	ID      types.String `tfsdk:"id"`
}

// ScimGroupRuleResource is a helper function to simplify the provider implementation.
func ScimGroupRuleResource() resource.Resource {
	return &scimGroupRuleResource{}
}

// scimGroupRuleResource is the resource implementation.
type scimGroupRuleResource struct {
	client *elves.Client
}

// Configure adds the provider configured client to the resource.
func (r *scimGroupRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*elves.Client)
}

// Metadata returns the resource type name.
func (r *scimGroupRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scim_group_rule"
}

// Schema defines the schema for the resource.
func (r *scimGroupRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Includes or excludes a rule for a SCIM group (for example an Okta group). " +
			"It applies to every Mac whose primary user is an active member of the group, after the Mac's configuration rules " +
			"and before SCIM user rules and rules set on the Mac itself. If a user's groups disagree on a rule, exclude wins.",
		Attributes: map[string]schema.Attribute{
			"group_id": schema.Int64Attribute{
				Required:    true,
				Description: "Required. ID of the SCIM group in Elves, for example `data.elves_scim_group.example.id`. Changing it replaces the assignment.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"rule_id": assignmentRuleIDAttribute(),
			"action":  assignmentActionAttribute(),
			"id":      assignmentIDAttribute("<group_id>/<rule_id>"),
		},
	}
}

func (r *scimGroupRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	groupStr, ruleStr, ok := strings.Cut(req.ID, "/")
	groupID, err1 := strconv.ParseInt(groupStr, 10, 64)
	ruleID, err2 := strconv.ParseInt(ruleStr, 10, 64)
	if !ok || err1 != nil || err2 != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Expected <group_id>/<rule_id>, got: "+req.ID)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_id"), groupID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("rule_id"), ruleID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// Create assigns the rule to the group
func (r *scimGroupRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan scimGroupRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID, ruleID := plan.GroupID.ValueInt64(), plan.RuleID.ValueInt64()
	if existing, err := r.client.ScimGroupRules(groupID); err == nil && findAssignedRule(existing, ruleID) != nil {
		resp.Diagnostics.AddError(
			"Rule already assigned to SCIM group",
			fmt.Sprintf("Rule %d is already assigned to SCIM group %d. Import it instead: terraform import <address> %d/%d", ruleID, groupID, groupID, ruleID),
		)
		return
	}
	if err := r.client.ScimGroupRuleAssign(groupID, ruleID, plan.Action.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error assigning rule to SCIM group",
			"Could not assign rule, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%d/%d", groupID, ruleID))
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *scimGroupRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state scimGroupRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rules, err := r.client.ScimGroupRules(state.GroupID.ValueInt64())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading SCIM group rules",
			"Could not read rules of SCIM group "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}
	rule := findAssignedRule(rules, state.RuleID.ValueInt64())
	if rule == nil {
		// Unassigned in the UI, or the group or rule was deleted
		resp.State.RemoveResource(ctx)
		return
	}

	state.Action = types.StringValue(rule.Action)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update changes the action; assigning again overwrites it
func (r *scimGroupRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan scimGroupRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.ScimGroupRuleAssign(plan.GroupID.ValueInt64(), plan.RuleID.ValueInt64(), plan.Action.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error updating SCIM group rule",
			"Could not update rule assignment, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *scimGroupRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state scimGroupRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.ScimGroupRuleUnassign(state.GroupID.ValueInt64(), state.RuleID.ValueInt64())
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError(
			"Error removing rule from SCIM group",
			"Could not remove rule assignment, unexpected error: "+err.Error(),
		)
	}
}
