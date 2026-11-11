package provider

import (
	"context"
	"regexp"
	"strconv"

	"github.com/DavidKrau/elves-go-client"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &faaRuleResource{}
	_ resource.ResourceWithConfigure   = &faaRuleResource{}
	_ resource.ResourceWithImportState = &faaRuleResource{}
)

// faaRuleTypes are the Options.RuleType values Santa accepts
var faaRuleTypes = []string{
	"PathsWithAllowedProcesses",
	"PathsWithDeniedProcesses",
	"ProcessesWithAllowedPaths",
	"ProcessesWithDeniedPaths",
}

type faaPathModel struct {
	Path     types.String `tfsdk:"path"`
	IsPrefix types.Bool   `tfsdk:"is_prefix"`
}

type faaProcessModel struct {
	SigningID         types.String `tfsdk:"signing_id"`
	TeamID            types.String `tfsdk:"team_id"`
	PlatformBinary    types.Bool   `tfsdk:"platform_binary"`
	CDHash            types.String `tfsdk:"cd_hash"`
	CertificateSha256 types.String `tfsdk:"certificate_sha256"`
	BinaryPath        types.String `tfsdk:"binary_path"`
}

// faaRuleResourceModel maps the resource schema data.
type faaRuleResourceModel struct {
	ID                        types.String      `tfsdk:"id"`
	Name                      types.String      `tfsdk:"name"`
	RuleType                  types.String      `tfsdk:"rule_type"`
	Paths                     []faaPathModel    `tfsdk:"paths"`
	Processes                 []faaProcessModel `tfsdk:"processes"`
	AllowReadAccess           types.Bool        `tfsdk:"allow_read_access"`
	AuditOnly                 types.Bool        `tfsdk:"audit_only"`
	ParentDirectoryProtection types.String      `tfsdk:"parent_directory_protection"`
	EnableSilentMode          types.Bool        `tfsdk:"enable_silent_mode"`
	EnableSilentTTYMode       types.Bool        `tfsdk:"enable_silent_tty_mode"`
	BlockMessage              types.String      `tfsdk:"block_message"`
	EventDetailURL            types.String      `tfsdk:"event_detail_url"`
	EventDetailText           types.String      `tfsdk:"event_detail_text"`
	Description               types.String      `tfsdk:"description"`
	Warnings                  types.List        `tfsdk:"warnings"`
}

// nullIfEmpty maps the API's empty strings to null, for optional attributes without a default
func nullIfEmpty(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

func (m faaRuleResourceModel) toAPI() elves.FAARule {
	rule := elves.FAARule{
		Name:            m.Name.ValueString(),
		RuleType:        m.RuleType.ValueString(),
		Paths:           []elves.FAAPath{},
		Processes:       []elves.FAAProcess{},
		AllowReadAccess: m.AllowReadAccess.ValueBool(),
		AuditOnly:       m.AuditOnly.ValueBool(),
		// Unknown (not set on create) is sent empty, so Elves picks the default
		ParentDirectoryProtection: m.ParentDirectoryProtection.ValueString(),
		EnableSilentMode:          m.EnableSilentMode.ValueBool(),
		EnableSilentTTYMode:       m.EnableSilentTTYMode.ValueBool(),
		BlockMessage:              m.BlockMessage.ValueString(),
		EventDetailURL:            m.EventDetailURL.ValueString(),
		EventDetailText:           m.EventDetailText.ValueString(),
		Description:               m.Description.ValueString(),
	}
	for _, p := range m.Paths {
		rule.Paths = append(rule.Paths, elves.FAAPath{Path: p.Path.ValueString(), IsPrefix: p.IsPrefix.ValueBool()})
	}
	for _, p := range m.Processes {
		process := elves.FAAProcess{
			SigningID:         p.SigningID.ValueString(),
			TeamID:            p.TeamID.ValueString(),
			CDHash:            p.CDHash.ValueString(),
			CertificateSha256: p.CertificateSha256.ValueString(),
			BinaryPath:        p.BinaryPath.ValueString(),
		}
		if !p.PlatformBinary.IsNull() && !p.PlatformBinary.IsUnknown() {
			pb := p.PlatformBinary.ValueBool()
			process.PlatformBinary = &pb
		}
		rule.Processes = append(rule.Processes, process)
	}
	return rule
}

func (m *faaRuleResourceModel) fromAPI(ctx context.Context, rule *elves.FAARule) diag.Diagnostics {
	m.ID = types.StringValue(strconv.FormatInt(rule.ID, 10))
	m.Name = types.StringValue(rule.Name)
	m.RuleType = types.StringValue(rule.RuleType)
	m.Paths = make([]faaPathModel, 0, len(rule.Paths))
	for _, p := range rule.Paths {
		m.Paths = append(m.Paths, faaPathModel{Path: types.StringValue(p.Path), IsPrefix: types.BoolValue(p.IsPrefix)})
	}
	m.Processes = make([]faaProcessModel, 0, len(rule.Processes))
	for _, p := range rule.Processes {
		process := faaProcessModel{
			SigningID:         nullIfEmpty(p.SigningID),
			TeamID:            nullIfEmpty(p.TeamID),
			PlatformBinary:    types.BoolNull(),
			CDHash:            nullIfEmpty(p.CDHash),
			CertificateSha256: nullIfEmpty(p.CertificateSha256),
			BinaryPath:        nullIfEmpty(p.BinaryPath),
		}
		if p.PlatformBinary != nil {
			process.PlatformBinary = types.BoolValue(*p.PlatformBinary)
		}
		m.Processes = append(m.Processes, process)
	}
	m.AllowReadAccess = types.BoolValue(rule.AllowReadAccess)
	m.AuditOnly = types.BoolValue(rule.AuditOnly)
	m.ParentDirectoryProtection = types.StringValue(rule.ParentDirectoryProtection)
	m.EnableSilentMode = types.BoolValue(rule.EnableSilentMode)
	m.EnableSilentTTYMode = types.BoolValue(rule.EnableSilentTTYMode)
	m.BlockMessage = types.StringValue(rule.BlockMessage)
	m.EventDetailURL = types.StringValue(rule.EventDetailURL)
	m.EventDetailText = types.StringValue(rule.EventDetailText)
	m.Description = types.StringValue(rule.Description)

	warnings := rule.Warnings
	if warnings == nil {
		warnings = []string{}
	}
	var diags diag.Diagnostics
	m.Warnings, diags = types.ListValueFrom(ctx, types.StringType, warnings)
	return diags
}

// addWarnings shows Elves' warnings about the rule, likely mistakes that Santa accepts
func addWarnings(rule *elves.FAARule, diags *diag.Diagnostics) {
	for _, w := range rule.Warnings {
		diags.AddWarning("Elves warning for FAA rule "+rule.Name, w)
	}
}

// FAARuleResource is a helper function to simplify the provider implementation.
func FAARuleResource() resource.Resource {
	return &faaRuleResource{}
}

// faaRuleResource is the resource implementation.
type faaRuleResource struct {
	client *elves.Client
}

// Configure adds the provider configured client to the resource.
func (r *faaRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*elves.Client)
}

// Metadata returns the resource type name.
func (r *faaRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_faa_rule"
}

// Schema defines the schema for the resource.
func (r *faaRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	optionalString := func(description string, validators ...validator.String) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString(""), Description: description, Validators: validators}
	}
	optionalBool := func(description string, def bool) schema.BoolAttribute {
		return schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(def), Description: description}
	}
	processString := func(description string, validators ...validator.String) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Description: description, Validators: append([]validator.String{stringvalidator.LengthAtLeast(1)}, validators...)}
	}
	resp.Schema = schema.Schema{
		Description: "Manages a Santa File Access Authorization (FAA) rule, one `WatchItems` entry of the FAA policy that Elves exports into the Santa MDM profile. " +
			"Every change bumps the policy version (see `elves_faa_settings`). " +
			"See https://northpole.dev/features/faa/ for how Santa applies the rules.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "ID of the FAA rule in Elves.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Required. Unique name of the rule, the `WatchItems` key in the policy. 1-64 letters, digits and `. _ : -`.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Za-z0-9._:-]{1,64}$`), "must be 1-64 characters of letters, digits and . _ : -"),
				},
			},
			"rule_type": schema.StringAttribute{
				Required: true,
				Description: "Required. `PathsWithAllowedProcesses` or `PathsWithDeniedProcesses` protect the paths from all but (or only from) the processes; " +
					"`ProcessesWithAllowedPaths` or `ProcessesWithDeniedPaths` limit the processes to (or keep them from) the paths.",
				Validators: []validator.String{stringvalidator.OneOf(faaRuleTypes...)},
			},
			"paths": schema.ListNestedAttribute{
				Required:    true,
				Description: "Required. Paths the rule watches. At least one.",
				Validators:  []validator.List{listvalidator.SizeAtLeast(1)},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"path": schema.StringAttribute{
							Required: true,
							Description: "Required. Absolute path, may use the globs `*`, `?` and `[]` but not `**`. " +
								"Santa doesn't follow symlinks, so use `/private/tmp`, `/private/var` and `/private/etc`.",
							Validators: []validator.String{stringvalidator.RegexMatches(regexp.MustCompile(`^/`), "must be an absolute path")},
						},
						"is_prefix": schema.BoolAttribute{
							Optional: true, Computed: true, Default: booldefault.StaticBool(false),
							Description: "Optional. Also match everything below the path. Defaults to false.",
						},
					},
				},
			},
			"processes": schema.ListNestedAttribute{
				Required: true,
				Description: "Required. Processes the rule applies to. At least one. A process matches only if every attribute that is set matches; " +
					"set at least one of them.",
				Validators: []validator.List{listvalidator.SizeAtLeast(1)},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"signing_id": processString("Optional. Signing ID, may end in `*`. A wildcard needs `team_id`, `platform_binary` or a `TEAMID:` or `platform:` prefix."),
						"team_id": processString("Optional. Team ID, 10 uppercase letters or digits.",
							stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Z0-9]{10}$`), "must be 10 uppercase letters or digits")),
						"platform_binary": schema.BoolAttribute{
							Optional:    true,
							Description: "Optional. Whether the process must be (true) or must not be (false) an Apple platform binary.",
						},
						"cd_hash": processString("Optional. CDHash, 40 hex characters.",
							stringvalidator.RegexMatches(regexp.MustCompile(`^[0-9a-fA-F]{40}$`), "must be 40 hex characters")),
						"certificate_sha256": processString("Optional. SHA-256 of the leaf signing certificate, 64 hex characters.",
							stringvalidator.RegexMatches(regexp.MustCompile(`^[0-9a-fA-F]{64}$`), "must be 64 hex characters")),
						"binary_path": processString("Optional. Absolute path of the binary, without globs.",
							stringvalidator.RegexMatches(regexp.MustCompile(`^/[^*?\[]*$`), "must be an absolute path without globs")),
					},
				},
			},
			"allow_read_access": optionalBool("Optional. Let every process read the paths, only writes are checked. Defaults to false.", false),
			"audit_only":        optionalBool("Optional. Only log accesses the rule would deny, as Santa does by default. Set to false to block them. Defaults to true.", true),
			"parent_directory_protection": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Optional. Whether Santa also guards renaming or deleting the paths' parent directories: `audit`, `enforce` or `disabled`. " +
					"Only used by `Paths*` rules. Elves sets `enforce` on new `Paths*` rules that leave it out.",
				Validators:    []validator.String{stringvalidator.OneOf("audit", "enforce", "disabled")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enable_silent_mode":     optionalBool("Optional. Don't show the block dialog on the Mac. Defaults to false.", false),
			"enable_silent_tty_mode": optionalBool("Optional. Don't print the block message in the terminal. Defaults to false.", false),
			"block_message":          optionalString("Optional. Message in the block dialog. Empty (the default) for Santa's message."),
			"event_detail_url": optionalString("Optional. URL the dialog's button opens, overrides the policy's (see `elves_faa_settings`). Must start with `https://`, may use Santa's placeholders. Empty (the default) for the policy's.",
				stringvalidator.RegexMatches(regexp.MustCompile(`^(https://.*)?$`), "must start with https://")),
			"event_detail_text": optionalString("Optional. Label of the dialog's button, at most 48 characters. Empty (the default) for the policy's.",
				stringvalidator.UTF8LengthAtMost(48)),
			"description": optionalString("Optional. Shown to users on the Elves FAA event page, not sent to Santa. Empty (the default) for none."),
			"warnings": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Likely mistakes Elves found in the rule that Santa accepts, for example a path that another rule watches too.",
			},
		},
	}
}

func (r *faaRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *faaRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan faaRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := plan.toAPI()
	rule, err := r.client.FAARuleCreate(&body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating FAA rule", "Could not create FAA rule, unexpected error: "+err.Error())
		return
	}
	addWarnings(rule, &resp.Diagnostics)

	resp.Diagnostics.Append(plan.fromAPI(ctx, rule)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *faaRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state faaRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := r.client.FAARuleGet(state.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading FAA rule", "Could not read FAA rule ID "+state.ID.ValueString()+": "+err.Error())
		return
	}

	resp.Diagnostics.Append(state.fromAPI(ctx, rule)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *faaRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan faaRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := plan.toAPI()
	rule, err := r.client.FAARuleUpdate(plan.ID.ValueString(), &body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating FAA rule", "Could not update FAA rule, unexpected error: "+err.Error())
		return
	}
	addWarnings(rule, &resp.Diagnostics)

	resp.Diagnostics.Append(plan.fromAPI(ctx, rule)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *faaRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state faaRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.FAARuleDelete(state.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Error deleting FAA rule", "Could not delete FAA rule, unexpected error: "+err.Error())
	}
}
