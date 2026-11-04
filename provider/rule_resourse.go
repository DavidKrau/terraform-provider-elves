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

	Description             types.String `tfsdk:"description"`
	RemediationInstructions types.String `tfsdk:"remediation_instructions"`
	RemediationScript       types.String `tfsdk:"remediation_script"`
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
		Description: "Manages an Elves (Santa) rule. `custommessage` is what Santa shows in its block dialog on the Mac; `description`, `remediation_instructions` and `remediation_script` are only shown on the Elves rules page that the dialog's button opens.",
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
				Description: "Required. Message Santa shows in the block dialog on the Mac when this rule blocks an app. Keep it short, it has to fit the dialog.",
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
			"description": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Optional. Longer explanation of why the app is blocked, shown on the Elves rules page that opens from the button in Santa's block dialog. Never sent to Santa. When empty, that page shows `custommessage` instead, so only set it when the page should say more than the dialog. When not set here, the value from the Elves UI is kept; set it to `\"\"` to clear it.",
			},
			"remediation_instructions": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Optional. What the user can do instead (for example an approved alternative or how to get access), shown on the Elves rules page below the description. Never sent to Santa. When not set here, the value from the Elves UI is kept; set it to `\"\"` to clear it.",
			},
			"remediation_script": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Script the user can copy and run in Terminal to fix the problem, shown on the Elves rules page. Read-only: Elves only lets a logged-in admin change it in the UI, not an API key, so Terraform keeps whatever is set there.",
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
		RuleType:                plan.RuleType.ValueString(),
		Policy:                  plan.Policy.ValueString(),
		Identifier:              plan.Identifier.ValueString(),
		CustomMsg:               plan.CustomMessage.ValueString(),
		CelExpr:                 plan.CelExpression.ValueString(),
		IsDefault:               plan.IsDefault.ValueBool(),
		Description:             plan.Description.ValueString(),
		RemediationInstructions: plan.RemediationInstructions.ValueString(),
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
	plan.Description = types.StringValue(rule.Description)
	plan.RemediationInstructions = types.StringValue(rule.RemediationInstructions)
	plan.RemediationScript = types.StringValue(rule.RemediationScript)

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
	state.Description = types.StringValue(rule.Description)
	state.RemediationInstructions = types.StringValue(rule.RemediationInstructions)
	state.RemediationScript = types.StringValue(rule.RemediationScript)

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

	// The update replaces the whole rule, and Elves refuses a changed script
	// from an API key, so send back the script it has now
	current, err := r.client.RuleGet(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating rule",
			"Could not read rule before updating it, unexpected error: "+err.Error(),
		)
		return
	}

	// Generate API request body from plan
	ruleUpdateRequest := elves.Rule{
		RuleType:                plan.RuleType.ValueString(),
		Policy:                  plan.Policy.ValueString(),
		Identifier:              plan.Identifier.ValueString(),
		CustomMsg:               plan.CustomMessage.ValueString(),
		CelExpr:                 plan.CelExpression.ValueString(),
		IsDefault:               plan.IsDefault.ValueBool(),
		Description:             plan.Description.ValueString(),
		RemediationInstructions: plan.RemediationInstructions.ValueString(),
		RemediationScript:       current.RemediationScript,
	}

	err = r.client.RuleUpdate(plan.ID.ValueString(), &ruleUpdateRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating rule",
			"Could not update rule, unexpected error: "+err.Error(),
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
