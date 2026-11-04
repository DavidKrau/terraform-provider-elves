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
	_ datasource.DataSource              = &scimGroupDataSource{}
	_ datasource.DataSourceWithConfigure = &scimGroupDataSource{}
)

// scimGroupDataSourceModel maps the data source schema data.
type scimGroupDataSourceModel struct {
	DisplayName types.String `tfsdk:"display_name"`
	MemberCount types.Int64  `tfsdk:"member_count"`
	ID          types.Int64  `tfsdk:"id"`
}

// ScimGroupDataSource is a helper function to simplify the provider implementation.
func ScimGroupDataSource() datasource.DataSource {
	return &scimGroupDataSource{}
}

// scimGroupDataSource is the data source implementation.
type scimGroupDataSource struct {
	client *elves.Client
}

// Metadata returns the data source type name.
func (d *scimGroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scim_group"
}

// Schema defines the schema for the data source.
func (d *scimGroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a SCIM group (for example an Okta group pushed to Elves) by its name, to get the ID that `elves_scim_group_rule` needs. Groups are created by SCIM provisioning, not by Terraform.",
		Attributes: map[string]schema.Attribute{
			"display_name": schema.StringAttribute{
				Required:    true,
				Description: "Required. Exact name of the group, as shown on the Elves SCIM Groups page.",
			},
			"id": schema.Int64Attribute{
				Computed:    true,
				Description: "ID of the group in Elves.",
			},
			"member_count": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of users in the group.",
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *scimGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state scimGroupDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groups, err := d.client.ScimGroups()
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Elves SCIM Groups", err.Error())
		return
	}
	var found *elves.ScimGroup
	for i := range groups {
		if groups[i].DisplayName != state.DisplayName.ValueString() {
			continue
		}
		if found != nil {
			resp.Diagnostics.AddError("Ambiguous SCIM group name", fmt.Sprintf("More than one SCIM group is named %q.", state.DisplayName.ValueString()))
			return
		}
		found = &groups[i]
	}
	if found == nil {
		resp.Diagnostics.AddError("SCIM group not found", fmt.Sprintf("No SCIM group is named %q. Check that it is pushed to Elves from your identity provider.", state.DisplayName.ValueString()))
		return
	}

	state.ID = types.Int64Value(found.ID)
	state.MemberCount = types.Int64Value(int64(found.MemberCount))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Configure adds the provider configured client to the data source.
func (d *scimGroupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
