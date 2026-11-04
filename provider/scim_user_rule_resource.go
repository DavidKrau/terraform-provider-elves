package provider

import (
	"context"

	"github.com/DavidKrau/elves-go-client"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &scimUserRuleResource{}
	_ resource.ResourceWithConfigure   = &scimUserRuleResource{}
	_ resource.ResourceWithImportState = &scimUserRuleResource{}
)

// scimUserRuleResourceModel maps the resource schema data.
type scimUserRuleResourceModel struct {
	RuleID types.Int64  `tfsdk:"rule_id"`
	Action types.String `tfsdk:"action"`
	Users  types.Set    `tfsdk:"users"`
	ID     types.String `tfsdk:"id"`
}

// ScimUserRuleResource is a helper function to simplify the provider implementation.
func ScimUserRuleResource() resource.Resource {
	return &scimUserRuleResource{}
}

// scimUserRuleResource is the resource implementation.
type scimUserRuleResource struct {
	client *elves.Client
}

// Configure adds the provider configured client to the resource.
func (r *scimUserRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*elves.Client)
}

// Metadata returns the resource type name.
func (r *scimUserRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scim_user_rule"
}

// Schema defines the schema for the resource.
func (r *scimUserRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Includes or excludes one rule for a list of SCIM users, matched by email. " +
			"A user's rules apply to every Mac whose primary user is that user, while the user is active. " +
			"They are applied after the Mac's configuration and SCIM group rules, and before rules set on the Mac itself. " +
			"Only the listed users are managed: other users that have the same rule (set in the UI or by another resource) are left alone.",
		Attributes: map[string]schema.Attribute{
			"rule_id": assignmentRuleIDAttribute(),
			"action":  assignmentActionAttribute(),
			"users": schema.SetAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "Required. Emails of the SCIM users, as provisioned from your identity provider. Matched against the user's email or user name, ignoring case. Every user must exist in Elves.",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the assignment, `<rule_id>/<action>`. Used for import, which brings in every user that has the rule with that action.",
			},
		},
	}
}

func (r *scimUserRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ruleID, action, err := parseMultiAssignmentID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("rule_id"), ruleID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("action"), action)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("users"), types.SetNull(types.StringType))...)
}

func (r *scimUserRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan scimUserRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	users := stringSetKeys(ctx, plan.Users, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := scimUserTargets.create(r.client, plan.RuleID.ValueInt64(), plan.Action.ValueString(), users); err != nil {
		resp.Diagnostics.AddError("Error assigning rule to SCIM users", err.Error())
		return
	}

	plan.ID = types.StringValue(multiAssignmentID(plan.RuleID.ValueInt64(), plan.Action.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *scimUserRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state scimUserRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	importing := state.Users.IsNull()
	users := stringSetKeys(ctx, state.Users, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := scimUserTargets.read(r.client, state.RuleID.ValueInt64(), state.Action.ValueString(), users, importing)
	if err != nil {
		resp.Diagnostics.AddError("Error reading SCIM user rules", "Could not read users of rule "+state.ID.ValueString()+": "+err.Error())
		return
	}
	if len(current) == 0 {
		// Removed from every user in the UI, or the rule was deleted
		resp.State.RemoveResource(ctx)
		return
	}

	state.Users = stringSet(current, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *scimUserRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state scimUserRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	oldUsers := stringSetKeys(ctx, state.Users, &resp.Diagnostics)
	newUsers := stringSetKeys(ctx, plan.Users, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := scimUserTargets.update(r.client, plan.RuleID.ValueInt64(), state.Action.ValueString(), plan.Action.ValueString(), oldUsers, newUsers); err != nil {
		resp.Diagnostics.AddError("Error updating SCIM user rules", err.Error())
		return
	}

	plan.ID = types.StringValue(multiAssignmentID(plan.RuleID.ValueInt64(), plan.Action.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *scimUserRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state scimUserRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	users := stringSetKeys(ctx, state.Users, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := scimUserTargets.remove(r.client, state.RuleID.ValueInt64(), state.Action.ValueString(), users); err != nil {
		resp.Diagnostics.AddError("Error removing rule from SCIM users", err.Error())
	}
}

// stringSetKeys returns the elements of a set of strings (none for a null set).
func stringSetKeys(ctx context.Context, set types.Set, diags *diag.Diagnostics) []string {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}
	var keys []string
	diags.Append(set.ElementsAs(ctx, &keys, false)...)
	return keys
}

func stringSet(keys []string, diags *diag.Diagnostics) types.Set {
	values := make([]attr.Value, len(keys))
	for i, k := range keys {
		values[i] = types.StringValue(k)
	}
	set, d := types.SetValue(types.StringType, values)
	diags.Append(d...)
	return set
}
