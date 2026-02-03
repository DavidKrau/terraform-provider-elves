package provider

import (
	"context"
	"strconv"
	"strings"

	"github.com/DavidKrau/elves-go-client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &ruleResource{}
	_ resource.ResourceWithConfigure   = &ruleResource{}
	_ resource.ResourceWithImportState = &ruleResource{}
)

// ruleResourceModel maps the resource schema data.
type ruleResourceModel struct {
	RuleType      types.String `tfsdk:"ruletype"`
	Policy        types.String `tfsdk:"policy"`
	Identifier    types.String `tfsdk:"identifier"`
	CelExpression types.String `tfsdk:"celexpression"`
	CustomMessage types.String `tfsdk:"custommessage"`
	IsDefault     types.Bool   `tfsdk:"isdefault"`
	ID            types.String `tfsdk:"id"`
}

// ruleResource is a helper function to simplify the provider implementation.
func RulesResource() resource.Resource {
	return &ruleResource{}
}

// ruleResource is the resource implementation.
type ruleResource struct {
	client *elves.Client
}

// Configure adds the provider configured client to the resource.
func (r *ruleResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*elves.Client)
}

// Metadata returns the resource type name.
func (r *ruleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rule"
}

// Schema defines the schema for the resource.
func (r *ruleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Rule resourse can be used to manage elves rule. Can be used together with configurations.",
		Attributes: map[string]schema.Attribute{
			"ruletype": schema.StringAttribute{
				Required:    true,
				Optional:    false,
				Description: "Required. BINARY, CERTIFICATE, TEAMID, SIGNINGID or CDHASH",
				Validators: []validator.String{
					stringvalidator.OneOf("BINARY", "CERTIFICATE", "TEAMID", "SIGNINGID", "CDHASH"),
				},
			},
			"policy": schema.StringAttribute{
				Optional:    false,
				Required:    true,
				Description: "Required. ALLOWLIST_COMPILER, ALLOWLIST, BLOCKLIST, SILENT_BLOCKLIST or CEL",
				Validators: []validator.String{
					stringvalidator.OneOf("ALLOWLIST_COMPILER", "ALLOWLIST", "BLOCKLIST", "SILENT_BLOCKLIST", "CEL"),
				},
			},
			"identifier": schema.StringAttribute{
				Optional:    false,
				Required:    true,
				Description: "Required. Identifier of the binary",
			},
			"celexpression": schema.StringAttribute{
				Optional:    true,
				Required:    false,
				Description: "Optiopnal. CEL expression",
			},
			"custommessage": schema.StringAttribute{
				Optional:    false,
				Required:    true,
				Description: "Required. Message shown to user if app is blocked.",
			},
			"isdefault": schema.BoolAttribute{
				Optional:    true,
				Default:     booldefault.StaticBool(true),
				Computed:    true,
				Description: "Optional. Marking rule and Default rule, default to true.",
			},
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "ID of a rule in Elves",
			},
		},
	}
}

func (r *ruleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id rule
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Create a new resource
func (r *ruleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	//Retrieve values from plan
	var plan ruleResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	ruleCreateRequest := elves.Rule{
		RuleType:   plan.RuleType.ValueString(),
		Policy:     plan.Policy.ValueString(),
		Identifier: plan.Identifier.ValueString(),
		CustomMsg:  plan.CustomMessage.ValueString(),
		CelExpr:    plan.CelExpression.ValueString(),
		IsDefault:  plan.IsDefault.ValueBool(),
	}
	rule, err := r.client.RuleCreate(&ruleCreateRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating rule",
			"Could not create rule, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(strconv.Itoa(int(rule.ID)))

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ruleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state ruleResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed rule value from elves
	rule, err := r.client.RuleGet(state.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading elves rule",
			"Could not read elves rule ID "+state.CustomMessage.ValueString()+": "+err.Error(),
		)
		return
	}

	// Overwrite items with refreshed state
	state.CustomMessage = types.StringValue(rule.CustomMsg)
	if rule.CelExpr != "" {
		state.CelExpression = types.StringValue(rule.CelExpr)
	}
	state.Identifier = types.StringValue(rule.Identifier)
	state.IsDefault = types.BoolValue(rule.IsDefault)
	state.Policy = types.StringValue(rule.Policy)
	state.RuleType = types.StringValue(rule.RuleType)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ruleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	//Retrieve values from plan
	var plan ruleResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	ruleUpdateRequest := elves.Rule{
		RuleType:   plan.RuleType.ValueString(),
		Policy:     plan.Policy.ValueString(),
		Identifier: plan.Identifier.ValueString(),
		CustomMsg:  plan.CustomMessage.ValueString(),
		CelExpr:    plan.CelExpression.ValueString(),
		IsDefault:  plan.IsDefault.ValueBool(),
	}

	// Generate API request body from plan
	err := r.client.RuleUpdate(plan.ID.ValueString(), &ruleUpdateRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating rule",
			"Could not create rule, unexpected error: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ruleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ruleResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing rule
	err := r.client.RuleDelete(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting elves rule",
			"Could not rule, unexpected error: "+err.Error(),
		)
		return
	}
}
