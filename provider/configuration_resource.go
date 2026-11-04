package provider

import (
	"context"
	"strconv"

	"github.com/DavidKrau/elves-go-client"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &configurationResource{}
	_ resource.ResourceWithConfigure   = &configurationResource{}
	_ resource.ResourceWithImportState = &configurationResource{}
)

// The default configuration is created by Elves and can't be changed over the API.
const defaultConfigurationID = "1"

// configurationResourceModel maps the resource schema data.
type configurationResourceModel struct {
	Name                     types.String `tfsdk:"name"`
	ClientMode               types.String `tfsdk:"client_mode"`
	SyncType                 types.String `tfsdk:"sync_type"`
	BlockedPathRegex         types.String `tfsdk:"blocked_path_regex"`
	AllowedPathRegex         types.String `tfsdk:"allowed_path_regex"`
	BatchSize                types.Int64  `tfsdk:"batch_size"`
	EnableBundles            types.Bool   `tfsdk:"enable_bundles"`
	EnableTransitiveRules    types.Bool   `tfsdk:"enable_transitive_rules"`
	FullSyncInterval         types.Int64  `tfsdk:"full_sync_interval"`
	BlockUSBMount            types.Bool   `tfsdk:"block_usb_mount"`
	RemountUSBMode           types.String `tfsdk:"remount_usb_mode"`
	OverrideFileAccessAction types.String `tfsdk:"override_file_access_action"`
	ID                       types.String `tfsdk:"id"`
}

func (m configurationResourceModel) toAPI() elves.Configuration {
	return elves.Configuration{
		Name:                     m.Name.ValueString(),
		ClientMode:               m.ClientMode.ValueString(),
		SyncType:                 m.SyncType.ValueString(),
		BlockedPathRegex:         m.BlockedPathRegex.ValueString(),
		AllowedPathRegex:         m.AllowedPathRegex.ValueString(),
		BatchSize:                m.BatchSize.ValueInt64(),
		EnableBundles:            m.EnableBundles.ValueBool(),
		EnableTransitiveRules:    m.EnableTransitiveRules.ValueBool(),
		FullSyncInterval:         m.FullSyncInterval.ValueInt64(),
		BlockUSBMount:            m.BlockUSBMount.ValueBool(),
		RemountUSBMode:           m.RemountUSBMode.ValueString(),
		OverrideFileAccessAction: m.OverrideFileAccessAction.ValueString(),
	}
}

func (m *configurationResourceModel) fromAPI(c *elves.Configuration) {
	m.Name = types.StringValue(c.Name)
	m.ClientMode = types.StringValue(c.ClientMode)
	m.SyncType = types.StringValue(c.SyncType)
	m.BlockedPathRegex = types.StringValue(c.BlockedPathRegex)
	m.AllowedPathRegex = types.StringValue(c.AllowedPathRegex)
	m.BatchSize = types.Int64Value(c.BatchSize)
	m.EnableBundles = types.BoolValue(c.EnableBundles)
	m.EnableTransitiveRules = types.BoolValue(c.EnableTransitiveRules)
	m.FullSyncInterval = types.Int64Value(c.FullSyncInterval)
	m.BlockUSBMount = types.BoolValue(c.BlockUSBMount)
	m.RemountUSBMode = types.StringValue(c.RemountUSBMode)
	m.OverrideFileAccessAction = types.StringValue(c.OverrideFileAccessAction)
}

// ConfigurationResource is a helper function to simplify the provider implementation.
func ConfigurationResource() resource.Resource {
	return &configurationResource{}
}

// configurationResource is the resource implementation.
type configurationResource struct {
	client *elves.Client
}

// Configure adds the provider configured client to the resource.
func (r *configurationResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*elves.Client)
}

// Metadata returns the resource type name.
func (r *configurationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_configuration"
}

// Schema defines the schema for the resource.
func (r *configurationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	optionalString := func(description string, def string, validators ...validator.String) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString(def), Description: description, Validators: validators}
	}
	optionalBool := func(description string, def bool) schema.BoolAttribute {
		return schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(def), Description: description}
	}
	resp.Schema = schema.Schema{
		Description: "Manages an Elves configuration: the Santa settings sent to every Mac using it, and the base for its rules (see `elves_config_rule`). " +
			"New Macs start on the default configuration (ID 1), which is managed by Elves and can't be changed or imported. " +
			"Deleting a configuration moves its Macs to the default configuration and removes its rule assignments; on their next sync the Macs get the default configuration's settings and rules.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Required. Name shown in Elves.",
			},
			"client_mode": schema.StringAttribute{
				Required:    true,
				Description: "Required. `MONITOR` only logs apps that would be blocked; `LOCKDOWN` blocks every app that no rule allows.",
				Validators:  []validator.String{stringvalidator.OneOf("MONITOR", "LOCKDOWN")},
			},
			"sync_type": optionalString(
				"Optional. How Macs apply the rule list: `CLEAN_ALL` replaces all rules on every sync, `CLEAN` replaces the rules from the sync server, `NORMAL` only adds and changes rules (removed rules stay on the Mac until a clean sync). Defaults to `CLEAN_ALL`.",
				"CLEAN_ALL", stringvalidator.OneOf("CLEAN_ALL", "CLEAN", "NORMAL")),
			"blocked_path_regex": optionalString("Optional. Regex of paths that are always blocked, whatever the rules say. Empty (the default) for none.", ""),
			"allowed_path_regex": optionalString("Optional. Regex of paths that are always allowed, whatever the rules say. Empty (the default) for none.", ""),
			"batch_size": schema.Int64Attribute{
				Optional: true, Computed: true, Default: int64default.StaticInt64(50000),
				Description: "Optional. Number of rules sent per rule download request. Defaults to 50000.",
				Validators:  []validator.Int64{int64validator.AtLeast(0)},
			},
			"enable_bundles":          optionalBool("Optional. Let Santa hash whole app bundles, so a bundle can be approved at once. Defaults to true.", true),
			"enable_transitive_rules": optionalBool("Optional. Allow binaries created by an `ALLOWLIST_COMPILER` rule. Defaults to false.", false),
			"full_sync_interval": schema.Int64Attribute{
				Optional: true, Computed: true, Default: int64default.StaticInt64(600),
				Description: "Optional. Seconds between full syncs. Defaults to 600.",
				Validators:  []validator.Int64{int64validator.AtLeast(0)},
			},
			"block_usb_mount":  optionalBool("Optional. Block mounting USB mass storage. Defaults to false.", false),
			"remount_usb_mode": optionalString("Optional. Mount flags for USB storage when `block_usb_mount` is on, for example `noexec,rdonly` to allow it read-only. Empty (the default) blocks it entirely.", ""),
			"override_file_access_action": optionalString(
				"Optional. Override for file access rules: `NONE` uses them as written, `AUDIT_ONLY` only logs, `DISABLE` turns them off. Defaults to `NONE`.",
				"NONE", stringvalidator.OneOf("NONE", "AUDIT_ONLY", "DISABLE")),
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "ID of the configuration in Elves, for example for `elves_config_rule`.",
			},
		},
	}
}

func (r *configurationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == defaultConfigurationID {
		resp.Diagnostics.AddError("Can't import the default configuration", "The default configuration (ID 1) is managed by Elves and can't be changed over the API.")
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *configurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan configurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := plan.toAPI()
	config, err := r.client.ConfigCreate(&body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating configuration", "Could not create configuration, unexpected error: "+err.Error())
		return
	}

	plan.ID = types.StringValue(strconv.FormatInt(config.ID, 10))
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *configurationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state configurationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, err := r.client.ConfigGet(state.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading configuration", "Could not read configuration ID "+state.ID.ValueString()+": "+err.Error())
		return
	}

	state.fromAPI(config)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *configurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan configurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := plan.toAPI()
	if err := r.client.ConfigUpdate(plan.ID.ValueString(), &body); err != nil {
		resp.Diagnostics.AddError("Error updating configuration", "Could not update configuration, unexpected error: "+err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *configurationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state configurationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.ConfigDelete(state.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Error deleting configuration", "Could not delete configuration, unexpected error: "+err.Error())
	}
}
