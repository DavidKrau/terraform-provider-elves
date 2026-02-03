package provider

import (
	"context"
	"os"

	"github.com/DavidKrau/elves-go-client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ provider.Provider = &elvesProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &elvesProvider{
			version: version,
		}
	}
}

// Provider is the provider implementation.
type elvesProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// ProviderModel maps provider schema data to a Go type.
type elvesProviderModel struct {
	Host         types.String `tfsdk:"host"`
	APIKeyName   types.String `tfsdk:"apikeyname"`
	APIKeySecret types.String `tfsdk:"apikeysecret"`
}

// Metadata returns the provider type name.
func (p *elvesProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "elves"
}

// Schema defines the provider-level schema for configuration data.
func (p *elvesProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Elves terraform provider developed by FreeNow.",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Optional:    true,
				Description: "API host for you instance, can be set as environment variable _HOST, if not set it will default to a..com",
			},
			"apikeyname": schema.StringAttribute{
				Optional:    true,
				Description: "API key for you instance, can be set as environment variable _APIKEY",
			},
			"apikeysecret": schema.StringAttribute{
				Optional:    true,
				Description: "API key for you instance, can be set as environment variable _APIKEY",
			},
		},
	}
}

func (p *elvesProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring  client")

	//Retrieve provider data from configuration
	var config elvesProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.
	if config.Host.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Unknown  Host",
			"The provider cannot create the  API client as there is an unknown configuration value for the  host. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the _HOST environment variable.",
		)
	}

	if config.APIKeyName.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("apikeyname"),
			"Unknown  API key",
			"The provider cannot create the  API client as there is an unknown configuration value for the  API key. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the _APIKEY environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.
	host := os.Getenv("_HOST")
	apikeyname := os.Getenv("_APIKEYNAME")
	apikeysecret := os.Getenv("_APIKEYSECRET")

	if !config.Host.IsNull() {
		host = config.Host.ValueString()
	}

	if !config.APIKeyName.IsNull() {
		apikeyname = config.APIKeyName.ValueString()
	}
	if !config.APIKeySecret.IsNull() {
		apikeysecret = config.APIKeySecret.ValueString()
	}

	if host == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Missing  host",
			"The provider cannot create the  API client as there is a missing or empty value for the  host. "+
				"Set the apikey value in the configuration or use the _HOST environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if apikeysecret == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("apikey"),
			"Missing  API key secret",
			"The provider cannot create the  API client as there is a missing or empty value for the  API key secret. "+
				"Set the apikey value in the configuration or use the _APIKEYSECRET environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if apikeyname == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("apikey"),
			"Missing  API key",
			"The provider cannot create the  API client as there is a missing or empty value for the  API key name. "+
				"Set the apikey value in the configuration or use the _APIKEYNAME environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "_host", host)
	ctx = tflog.SetField(ctx, "_apikeyname", apikeyname)
	ctx = tflog.SetField(ctx, "_apikeyname", apikeysecret)

	tflog.Debug(ctx, "Creating  client")

	apiClient := elves.NewClient(host, apikeyname, apikeysecret)

	// Make the  client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = apiClient
	resp.ResourceData = apiClient

	tflog.Info(ctx, "Configured  client", map[string]any{"success": true})

}

// DataSources defines the data sources implemented in the provider.
func (p *elvesProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		RulesDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *elvesProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		RulesResource,
	}
}
