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
	_ datasource.DataSource              = &configurationDataSource{}
	_ datasource.DataSourceWithConfigure = &configurationDataSource{}
)

// configurationDataSourceModel maps the data source schema data.
type configurationDataSourceModel struct {
	Name       types.String `tfsdk:"name"`
	ClientMode types.String `tfsdk:"client_mode"`
	SyncType   types.String `tfsdk:"sync_type"`
	ID         types.Int64  `tfsdk:"id"`
}

// ConfigurationDataSource is a helper function to simplify the provider implementation.
func ConfigurationDataSource() datasource.DataSource {
	return &configurationDataSource{}
}

// configurationDataSource is the data source implementation.
type configurationDataSource struct {
	client *elves.Client
}

// Metadata returns the data source type name.
func (d *configurationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_configuration"
}

// Schema defines the schema for the data source.
func (d *configurationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a configuration by its name, for example the default configuration or one created in the UI, to get the ID that `elves_config_rule` needs.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Required. Exact name of the configuration, as shown on the Elves Configurations page.",
			},
			"id": schema.Int64Attribute{
				Computed:    true,
				Description: "ID of the configuration in Elves.",
			},
			"client_mode": schema.StringAttribute{
				Computed:    true,
				Description: "`MONITOR` or `LOCKDOWN`.",
			},
			"sync_type": schema.StringAttribute{
				Computed:    true,
				Description: "`CLEAN_ALL`, `CLEAN` or `NORMAL`.",
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *configurationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state configurationDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	configs, err := d.client.Configs()
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Elves Configurations", err.Error())
		return
	}
	var found *elves.Configuration
	for i := range configs {
		if configs[i].Name != state.Name.ValueString() {
			continue
		}
		if found != nil {
			resp.Diagnostics.AddError("Ambiguous configuration name", fmt.Sprintf("More than one configuration is named %q.", state.Name.ValueString()))
			return
		}
		found = &configs[i]
	}
	if found == nil {
		resp.Diagnostics.AddError("Configuration not found", fmt.Sprintf("No configuration is named %q.", state.Name.ValueString()))
		return
	}

	state.ID = types.Int64Value(found.ID)
	state.ClientMode = types.StringValue(found.ClientMode)
	state.SyncType = types.StringValue(found.SyncType)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Configure adds the provider configured client to the data source.
func (d *configurationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
