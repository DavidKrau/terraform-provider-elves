package provider

import (
	"context"
	"fmt"

	"github.com/DavidKrau/elves-go-client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &ruleDataSource{}
	_ datasource.DataSourceWithConfigure = &ruleDataSource{}
)

// attributeDataSourceModel maps the data source schema data.
type ruleDataSourceModel struct {
	RuleType      types.String `tfsdk:"ruletype"`
	Policy        types.String `tfsdk:"policy"`
	Identifier    types.String `tfsdk:"identifier"`
	CelExpression types.String `tfsdk:"celexpression"`
	CustomMessage types.String `tfsdk:"custommessage"`
	IsDefault     types.Bool   `tfsdk:"isdefault"`
	ID            types.String `tfsdk:"id"`
}

// AttributeDataSource is a helper function to simplify the provider implementation.
func RulesDataSource() datasource.DataSource {
	return &ruleDataSource{}
}

// AttributeDataSource is the data source implementation.
type ruleDataSource struct {
	client *elves.Client
}

// Metadata returns the data source type name.
func (d *ruleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rule"
}

// Schema defines the schema for the data source.
func (d *ruleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Rule resourse can be used to manage elves rule. Can be used together with configurations.",
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
				Computed:    true,
				Description: "Optional. Marking rule and Default rule, default to true.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of a rule in Elves",
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *ruleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ruleDataSourceModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	attribute, err := d.client.RuleGet(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Elves Santa Rule",
			err.Error(),
		)
		return
	}

	// Map response body to model
	state.Identifier = types.StringValue(attribute.Identifier)
	state.RuleType = types.StringValue(attribute.RuleType)

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
