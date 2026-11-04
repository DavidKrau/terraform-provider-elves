package provider

import (
	"context"
	"fmt"

	"github.com/DavidKrau/elves-go-client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &ruleDataSource{}
	_ datasource.DataSourceWithConfigure = &ruleDataSource{}
)

// ruleDataSourceModel maps the data source schema data.
type ruleDataSourceModel struct {
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

// RulesDataSource is a helper function to simplify the provider implementation.
func RulesDataSource() datasource.DataSource {
	return &ruleDataSource{}
}

// ruleDataSource is the data source implementation.
type ruleDataSource struct {
	client *elves.Client
}

// Metadata returns the data source type name.
func (d *ruleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rule"
}

// Schema defines the schema for the data source.
func (d *ruleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Rule data source reads an existing elves rule by its ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "Required. ID of a rule in Elves",
			},
			"ruletype": schema.StringAttribute{
				Computed:    true,
				Description: "BINARY, CERTIFICATE, TEAMID, SIGNINGID or CDHASH",
			},
			"policy": schema.StringAttribute{
				Computed:    true,
				Description: "ALLOWLIST_COMPILER, ALLOWLIST, BLOCKLIST, SILENT_BLOCKLIST or CEL",
			},
			"identifier": schema.StringAttribute{
				Computed:    true,
				Description: "Identifier of the binary",
			},
			"celexpression": schema.StringAttribute{
				Computed:    true,
				Description: "CEL expression",
			},
			"custommessage": schema.StringAttribute{
				Computed:    true,
				Description: "Message Santa shows in the block dialog on the Mac when this rule blocks an app. Keep it short, it has to fit the dialog.",
			},
			"isdefault": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the rule is a Default rule.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Longer explanation of why the app is blocked, shown on the Elves rules page that opens from the button in Santa's block dialog. Never sent to Santa. When empty, that page shows `custommessage` instead, so only set it when the page should say more than the dialog.",
			},
			"remediation_instructions": schema.StringAttribute{
				Computed:    true,
				Description: "What the user can do instead (for example an approved alternative or how to get access), shown on the Elves rules page below the description. Never sent to Santa.",
			},
			"remediation_script": schema.StringAttribute{
				Computed:    true,
				Description: "Script the user can copy and run in Terminal to fix the problem, shown on the Elves rules page.",
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *ruleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ruleDataSourceModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := d.client.RuleGet(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Elves Santa Rule",
			err.Error(),
		)
		return
	}

	// Map response body to model
	state.RuleType = types.StringValue(rule.RuleType)
	state.Policy = types.StringValue(rule.Policy)
	state.Identifier = types.StringValue(rule.Identifier)
	state.CelExpression = types.StringValue(rule.CelExpr)
	state.CustomMessage = types.StringValue(rule.CustomMsg)
	state.IsDefault = types.BoolValue(rule.IsDefault)
	state.Description = types.StringValue(rule.Description)
	state.RemediationInstructions = types.StringValue(rule.RemediationInstructions)
	state.RemediationScript = types.StringValue(rule.RemediationScript)

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Configure adds the provider configured client to the data source.
func (d *ruleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*elves.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}
