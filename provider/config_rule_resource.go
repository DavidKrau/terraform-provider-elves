package provider

import (
	"context"
	"strconv"

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
	_ resource.Resource                = &configRuleResource{}
	_ resource.ResourceWithConfigure   = &configRuleResource{}
	_ resource.ResourceWithImportState = &configRuleResource{}
)

// configRuleResourceModel maps the resource schema data.
type configRuleResourceModel struct {
	RuleID  types.Int64  `tfsdk:"rule_id"`
	Action  types.String `tfsdk:"action"`
	Configs types.Set    `tfsdk:"config_ids"`
	ID      types.String `tfsdk:"id"`
}

// ConfigRuleResource is a helper function to simplify the provider implementation.
func ConfigRuleResource() resource.Resource {
	return &configRuleResource{}
}

// configRuleResource is the resource implementation.
type configRuleResource struct {
	client *elves.Client
}

// Configure adds the provider configured client to the resource.
func (r *configRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*elves.Client)
}

// Metadata returns the resource type name.
func (r *configRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config_rule"
}

// Schema defines the schema for the resource.
func (r *configRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Includes or excludes one rule for a list of configurations. " +
			"Configuration rules apply to every Mac using the configuration, after the default rules and before SCIM group, SCIM user and Mac rules. " +
			"Only the listed configurations are managed: other configurations that have the same rule (set in the UI or by another resource) are left alone.",
		Attributes: map[string]schema.Attribute{
			"rule_id": assignmentRuleIDAttribute(),
			"action":  assignmentActionAttribute(),
			"config_ids": schema.SetAttribute{
				ElementType: types.Int64Type,
				Required:    true,
				Description: "Required. IDs of the configurations, as shown on the Elves Configurations page.",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the assignment, `<rule_id>/<action>`. Used for import, which brings in every configuration that has the rule with that action.",
			},
		},
	}
}

func (r *configRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ruleID, action, err := parseMultiAssignmentID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("rule_id"), ruleID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("action"), action)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("config_ids"), types.SetNull(types.Int64Type))...)
}

func (r *configRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan configRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	configs := int64SetKeys(ctx, plan.Configs, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := configTargets.create(r.client, plan.RuleID.ValueInt64(), plan.Action.ValueString(), configs); err != nil {
		resp.Diagnostics.AddError("Error assigning rule to configurations", err.Error())
		return
	}

	plan.ID = types.StringValue(multiAssignmentID(plan.RuleID.ValueInt64(), plan.Action.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *configRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state configRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	importing := state.Configs.IsNull()
	configs := int64SetKeys(ctx, state.Configs, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := configTargets.read(r.client, state.RuleID.ValueInt64(), state.Action.ValueString(), configs, importing)
	if err != nil {
		resp.Diagnostics.AddError("Error reading configuration rules", "Could not read configurations of rule "+state.ID.ValueString()+": "+err.Error())
		return
	}
	if len(current) == 0 {
		// Removed from every configuration in the UI, or the rule was deleted
		resp.State.RemoveResource(ctx)
		return
	}

	state.Configs = int64Set(current, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *configRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state configRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	oldConfigs := int64SetKeys(ctx, state.Configs, &resp.Diagnostics)
	newConfigs := int64SetKeys(ctx, plan.Configs, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := configTargets.update(r.client, plan.RuleID.ValueInt64(), state.Action.ValueString(), plan.Action.ValueString(), oldConfigs, newConfigs); err != nil {
		resp.Diagnostics.AddError("Error updating configuration rules", err.Error())
		return
	}

	plan.ID = types.StringValue(multiAssignmentID(plan.RuleID.ValueInt64(), plan.Action.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *configRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state configRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	configs := int64SetKeys(ctx, state.Configs, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := configTargets.remove(r.client, state.RuleID.ValueInt64(), state.Action.ValueString(), configs); err != nil {
		resp.Diagnostics.AddError("Error removing rule from configurations", err.Error())
	}
}

// int64SetKeys returns the elements of a set of numbers as string keys.
func int64SetKeys(ctx context.Context, set types.Set, diags *diag.Diagnostics) []string {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}
	var ids []int64
	diags.Append(set.ElementsAs(ctx, &ids, false)...)
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = strconv.FormatInt(id, 10)
	}
	return keys
}

func int64Set(keys []string, diags *diag.Diagnostics) types.Set {
	values := make([]attr.Value, 0, len(keys))
	for _, k := range keys {
		id, err := strconv.ParseInt(k, 10, 64)
		if err != nil {
			diags.AddError("Invalid configuration ID", k)
			continue
		}
		values = append(values, types.Int64Value(id))
	}
	set, d := types.SetValue(types.Int64Type, values)
	diags.Append(d...)
	return set
}
