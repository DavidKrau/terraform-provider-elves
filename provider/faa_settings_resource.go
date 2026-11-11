package provider

import (
	"context"
	"regexp"

	"github.com/DavidKrau/elves-go-client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &faaSettingsResource{}
	_ resource.ResourceWithConfigure   = &faaSettingsResource{}
	_ resource.ResourceWithImportState = &faaSettingsResource{}
)

// The FAA settings are a single object in Elves, so they always have this ID
const faaSettingsID = "faa_settings"

// faaSettingsResourceModel maps the resource schema data.
type faaSettingsResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Version         types.String `tfsdk:"version"`
	EventDetailURL  types.String `tfsdk:"event_detail_url"`
	EventDetailText types.String `tfsdk:"event_detail_text"`
}

func (m *faaSettingsResourceModel) fromAPI(s *elves.FAASettings) {
	m.ID = types.StringValue(faaSettingsID)
	m.Version = types.StringValue(s.Version)
	m.EventDetailURL = types.StringValue(s.EventDetailURL)
	m.EventDetailText = types.StringValue(s.EventDetailText)
}

// FAASettingsResource is a helper function to simplify the provider implementation.
func FAASettingsResource() resource.Resource {
	return &faaSettingsResource{}
}

// faaSettingsResource is the resource implementation.
type faaSettingsResource struct {
	client *elves.Client
}

// Configure adds the provider configured client to the resource.
func (r *faaSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*elves.Client)
}

// Metadata returns the resource type name.
func (r *faaSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_faa_settings"
}

// Schema defines the schema for the resource.
func (r *faaSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the settings of the Santa File Access Authorization (FAA) policy that Elves exports into the Santa MDM profile. " +
			"Elves has exactly one set of FAA settings, so declare this resource at most once. " +
			"Destroying it only removes it from the Terraform state and leaves the settings in Elves as they are.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "Always `" + faaSettingsID + "`.",
			},
			"version": schema.StringAttribute{
				Computed: true,
				Description: "The policy version, read only. Elves increments it on every change to an FAA rule or these settings, " +
					"so Santa reloads the policy.",
			},
			"event_detail_url": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Optional. URL the block dialog's button opens, for rules that don't set their own. Must start with `https://`, " +
					"may use Santa's placeholders. Elves defaults it to its FAA event page; left out, the current value is kept. Set to empty for no button.",
				Validators:    []validator.String{stringvalidator.RegexMatches(regexp.MustCompile(`^(https://.*)?$`), "must start with https://")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"event_detail_text": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Optional. Label of the block dialog's button, at most 48 characters, for rules that don't set their own. Left out, the current value is kept.",
				Validators:    []validator.String{stringvalidator.UTF8LengthAtMost(48)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

// ImportState accepts any ID, there is only one set of settings.
func (r *faaSettingsResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), faaSettingsID)...)
}

// save writes the plan, keeping the current value of attributes left out
func (r *faaSettingsResource) save(plan *faaSettingsResourceModel) error {
	current, err := r.client.FAASettingsGet()
	if err != nil {
		return err
	}
	settings := *current
	if !plan.EventDetailURL.IsUnknown() {
		settings.EventDetailURL = plan.EventDetailURL.ValueString()
	}
	if !plan.EventDetailText.IsUnknown() {
		settings.EventDetailText = plan.EventDetailText.ValueString()
	}
	// The current version, so Elves bumps it
	saved, err := r.client.FAASettingsUpdate(&settings)
	if err != nil {
		return err
	}
	plan.fromAPI(saved)
	return nil
}

func (r *faaSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan faaSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.save(&plan); err != nil {
		resp.Diagnostics.AddError("Error saving FAA settings", "Could not save FAA settings, unexpected error: "+err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *faaSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state faaSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	settings, err := r.client.FAASettingsGet()
	if err != nil {
		resp.Diagnostics.AddError("Error reading FAA settings", "Could not read FAA settings: "+err.Error())
		return
	}
	state.fromAPI(settings)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *faaSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan faaSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.save(&plan); err != nil {
		resp.Diagnostics.AddError("Error saving FAA settings", "Could not save FAA settings, unexpected error: "+err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Delete leaves the settings in Elves, they can't be deleted.
func (r *faaSettingsResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}
