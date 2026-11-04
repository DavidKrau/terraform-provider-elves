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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &clientRuleResource{}
	_ resource.ResourceWithConfigure   = &clientRuleResource{}
	_ resource.ResourceWithImportState = &clientRuleResource{}
)

// clientRuleResourceModel maps the resource schema data.
type clientRuleResourceModel struct {
	SerialNumber types.String `tfsdk:"serial_number"`
	RuleID       types.Int64  `tfsdk:"rule_id"`
	Action       types.String `tfsdk:"action"`
	ID           types.String `tfsdk:"id"`
}

// ClientRuleResource is a helper function to simplify the provider implementation.
func ClientRuleResource() resource.Resource {
	return &clientRuleResource{}
}

// clientRuleResource is the resource implementation.
type clientRuleResource struct {
	client *elves.Client
}

// Configure adds the provider configured client to the resource.
func (r *clientRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*elves.Client)
}

// Metadata returns the resource type name.
func (r *clientRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_client_rule"
}

// Schema defines the schema for the resource.
func (r *clientRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Includes or excludes a rule for one Mac (client), by serial number. " +
			"Rules set on the Mac itself are applied last, after its configuration, SCIM group and SCIM user rules, so they win over all of them.",
		Attributes: map[string]schema.Attribute{
			"serial_number": schema.StringAttribute{
				Required:    true,
				Description: "Required. Serial number of the Mac, as shown on the Elves Clients page. The Mac must have synced with Elves at least once. Changing it replaces the assignment.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"rule_id": assignmentRuleIDAttribute(),
			"action":  assignmentActionAttribute(),
			"id":      assignmentIDAttribute("<serial_number>/<rule_id>"),
		},
	}
}

func (r *clientRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	serial, ruleStr, ok := strings.Cut(req.ID, "/")
	ruleID, err := strconv.ParseInt(ruleStr, 10, 64)
	if !ok || serial == "" || err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Expected <serial_number>/<rule_id>, got: "+req.ID)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("serial_number"), serial)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("rule_id"), ruleID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// Create assigns the rule to the client
func (r *clientRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan clientRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serial, ruleID := plan.SerialNumber.ValueString(), plan.RuleID.ValueInt64()
	if existing, err := r.client.ClientRules(serial); err == nil && findAssignedRule(existing, ruleID) != nil {
		resp.Diagnostics.AddError(
			"Rule already assigned to client",
			fmt.Sprintf("Rule %d is already assigned to client %s. Import it instead: terraform import <address> %s/%d", ruleID, serial, serial, ruleID),
		)
		return
	}
	if err := r.client.ClientRuleAssign(serial, ruleID, plan.Action.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error assigning rule to client",
			"Could not assign rule, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/%d", serial, ruleID))
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *clientRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state clientRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rules, err := r.client.ClientRules(state.SerialNumber.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading client rules",
			"Could not read rules of client "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}
	rule := findAssignedRule(rules, state.RuleID.ValueInt64())
	if rule == nil {
		// Unassigned in the UI, or the client or rule was deleted
		resp.State.RemoveResource(ctx)
		return
	}

	state.Action = types.StringValue(rule.Action)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update changes the action; assigning again overwrites it
func (r *clientRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan clientRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.ClientRuleAssign(plan.SerialNumber.ValueString(), plan.RuleID.ValueInt64(), plan.Action.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error updating client rule",
			"Could not update rule assignment, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *clientRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state clientRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.ClientRuleUnassign(state.SerialNumber.ValueString(), state.RuleID.ValueInt64())
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError(
			"Error removing rule from client",
			"Could not remove rule assignment, unexpected error: "+err.Error(),
		)
	}
}
