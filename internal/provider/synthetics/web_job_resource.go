package synthetics

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
	"github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/synthetics"
)

var (
	_ resource.Resource                   = &webJobResource{}
	_ resource.ResourceWithConfigure      = &webJobResource{}
	_ resource.ResourceWithImportState    = &webJobResource{}
	_ resource.ResourceWithValidateConfig = &webJobResource{}
)

func NewWebJobResource() resource.Resource {
	return &webJobResource{}
}

type webJobResource struct {
	client *client.Client
}

type webJobResourceModel struct {
	ID                      types.String         `tfsdk:"id"`
	ApplicationID           types.Int64          `tfsdk:"application_id"`
	AppKey                  types.String         `tfsdk:"app_key"`
	Description             types.String         `tfsdk:"description"`
	URL                     types.String         `tfsdk:"url"`
	ScriptJSON              jsontypes.Normalized `tfsdk:"script_json"`
	BrowserCodes            types.List           `tfsdk:"browser_codes"`
	ChromeVersions          types.List           `tfsdk:"chrome_versions"`
	LocationCodes           types.List           `tfsdk:"location_codes"`
	TimeoutSeconds          types.Int64          `tfsdk:"timeout_seconds"`
	UserEnabled             types.Bool           `tfsdk:"user_enabled"`
	ScheduleRunConfigsJSON  jsontypes.Normalized `tfsdk:"schedule_run_configs_json"`
	NetworkProfileJSON      jsontypes.Normalized `tfsdk:"network_profile_json"`
	PerformanceCriteriaJSON jsontypes.Normalized `tfsdk:"performance_criteria_json"`
	ComposableConfigJSON    jsontypes.Normalized `tfsdk:"composable_config_json"`
	Version                 types.Int64          `tfsdk:"version"`
}

func (r *webJobResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_synthetic_web_job"
}

func (r *webJobResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An AppDynamics Synthetic Web Monitoring job: a scheduled browser check (simple URL load or a Selenium script) against a Browser RUM (EUM) application. Uses the Controller's own internal API (verified live), since the officially documented Synthetic Monitoring API requires a separate EUM account username/license key credential pair this provider does not use.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"application_id": schema.Int64Attribute{
				Required:      true,
				Description:   "ID of the business application the Browser RUM app is associated with (visible in the Controller UI URL for that Browser App).",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"app_key": schema.StringAttribute{
				Required:      true,
				Description:   "EUM Browser App key (from User Experience > Browser Apps in the Controller UI) -- distinct from application_id.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"description": schema.StringAttribute{
				Required:    true,
				Description: "Job name, shown in the Controller UI's Jobs list.",
			},
			"url": schema.StringAttribute{
				Optional:    true,
				Description: "URL for a simple page-load check. Exactly one of url / script_json must be set.",
			},
			"script_json": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Optional:    true,
				Description: "JSON object {\"contentType\":\"INLINE_PYTHON_3\",\"script\":\"...\"} for a scripted (Selenium) check. Exactly one of url / script_json must be set.",
			},
			"browser_codes": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Browser codes to run the check from, e.g. [\"Chrome\"].",
			},
			"chrome_versions": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Chrome versions to run the check with, e.g. [\"86\"].",
			},
			"location_codes": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Synthetic monitoring location codes, e.g. [\"M50\"].",
			},
			"timeout_seconds": schema.Int64Attribute{
				Required:    true,
				Description: "Job timeout in seconds (5-300 per the docs).",
			},
			"user_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the job actively runs on its schedule. Defaults to false (verified live: the API leaves a newly created job disabled unless this is explicitly set true).",
			},
			"schedule_run_configs_json": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Required:    true,
				Description: "JSON array of run-schedule configs, e.g. [{\"rate\":{\"value\":15,\"unit\":\"MINUTES\"},\"daysOfWeek\":[\"MON\"],\"timezone\":\"UTC\"}]. rate.value must be 1-59 for MINUTES (verified live).",
			},
			"network_profile_json": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Optional:    true,
				Description: "JSON object describing simulated network conditions. See the Splunk AppDynamics Synthetic Monitoring API docs for the shape.",
			},
			"performance_criteria_json": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Optional:    true,
				Description: "JSON object describing pass/warn/critical performance thresholds. See the Splunk AppDynamics Synthetic Monitoring API docs for the shape.",
			},
			"composable_config_json": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Optional:    true,
				Description: "JSON object describing resource-error-detection rules. See the Splunk AppDynamics Synthetic Monitoring API docs for the shape.",
			},
			"version": schema.Int64Attribute{
				Computed:    true,
				Description: "Optimistic-locking version, incremented by the API on every update.",
			},
		},
	}
}

func (r *webJobResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data webJobResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasURL := !data.URL.IsNull() && !data.URL.IsUnknown()
	hasScript := !data.ScriptJSON.IsNull() && !data.ScriptJSON.IsUnknown()
	if hasURL == hasScript {
		resp.Diagnostics.AddError(
			"Invalid Attribute Combination",
			"Exactly one of \"url\" or \"script_json\" must be set.",
		)
	}
}

func (r *webJobResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func apiWebJobFromModel(ctx context.Context, m webJobResourceModel) (*synthetics.WebJob, diag.Diagnostics) {
	var diags diag.Diagnostics

	browserCodes, d := stringListToGo(ctx, m.BrowserCodes)
	diags.Append(d...)
	chromeVersions, d := stringListToGo(ctx, m.ChromeVersions)
	diags.Append(d...)
	locationCodes, d := stringListToGo(ctx, m.LocationCodes)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	job := &synthetics.WebJob{
		Description:        m.Description.ValueString(),
		AppKey:             m.AppKey.ValueString(),
		URL:                m.URL.ValueString(),
		BrowserCodes:       browserCodes,
		ChromeVersions:     chromeVersions,
		LocationCodes:      locationCodes,
		TimeoutSeconds:     int(m.TimeoutSeconds.ValueInt64()),
		UserEnabled:        m.UserEnabled.ValueBool(),
		ScheduleRunConfigs: json.RawMessage(m.ScheduleRunConfigsJSON.ValueString()),
	}
	if !m.ScriptJSON.IsNull() && !m.ScriptJSON.IsUnknown() {
		job.Script = json.RawMessage(m.ScriptJSON.ValueString())
	}
	if !m.NetworkProfileJSON.IsNull() && !m.NetworkProfileJSON.IsUnknown() {
		job.NetworkProfile = json.RawMessage(m.NetworkProfileJSON.ValueString())
	}
	if !m.PerformanceCriteriaJSON.IsNull() && !m.PerformanceCriteriaJSON.IsUnknown() {
		job.PerformanceCriteria = json.RawMessage(m.PerformanceCriteriaJSON.ValueString())
	}
	if !m.ComposableConfigJSON.IsNull() && !m.ComposableConfigJSON.IsUnknown() {
		job.ComposableConfig = json.RawMessage(m.ComposableConfigJSON.ValueString())
	}
	return job, diags
}

// hasJSON reports whether raw carries a real value rather than being absent
// or the literal JSON null (verified live: the API echoes optional blocks
// like composableConfig/performanceCriteria back as an explicit JSON null
// when unset, not by omitting the field).
func hasJSON(raw json.RawMessage) bool {
	return len(raw) > 0 && !bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}

func modelFromAPIWebJob(ctx context.Context, applicationID int64, job *synthetics.WebJob) (webJobResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	browserCodes, d := goStringList(ctx, job.BrowserCodes)
	diags.Append(d...)
	chromeVersions, d := goStringList(ctx, job.ChromeVersions)
	diags.Append(d...)
	locationCodes, d := goStringList(ctx, job.LocationCodes)
	diags.Append(d...)

	m := webJobResourceModel{
		ID:             types.StringValue(job.ID),
		ApplicationID:  types.Int64Value(applicationID),
		AppKey:         types.StringValue(job.AppKey),
		Description:    types.StringValue(job.Description),
		URL:            stringOrNull(job.URL),
		BrowserCodes:   browserCodes,
		ChromeVersions: chromeVersions,
		LocationCodes:  locationCodes,
		TimeoutSeconds: types.Int64Value(int64(job.TimeoutSeconds)),
		UserEnabled:    types.BoolValue(job.UserEnabled),
		Version:        types.Int64Value(int64(job.Version)),
	}
	if hasJSON(job.Script) {
		m.ScriptJSON = jsontypes.NewNormalizedValue(string(job.Script))
	} else {
		m.ScriptJSON = jsontypes.NewNormalizedNull()
	}
	if hasJSON(job.ScheduleRunConfigs) {
		m.ScheduleRunConfigsJSON = jsontypes.NewNormalizedValue(string(job.ScheduleRunConfigs))
	} else {
		m.ScheduleRunConfigsJSON = jsontypes.NewNormalizedNull()
	}
	if hasJSON(job.NetworkProfile) {
		m.NetworkProfileJSON = jsontypes.NewNormalizedValue(string(job.NetworkProfile))
	} else {
		m.NetworkProfileJSON = jsontypes.NewNormalizedNull()
	}
	if hasJSON(job.PerformanceCriteria) {
		m.PerformanceCriteriaJSON = jsontypes.NewNormalizedValue(string(job.PerformanceCriteria))
	} else {
		m.PerformanceCriteriaJSON = jsontypes.NewNormalizedNull()
	}
	if hasJSON(job.ComposableConfig) {
		m.ComposableConfigJSON = jsontypes.NewNormalizedValue(string(job.ComposableConfig))
	} else {
		m.ComposableConfigJSON = jsontypes.NewNormalizedNull()
	}
	return m, diags
}

// preserveJSONFromPlan keeps the plan's/prior state's own value for every
// JSON-passthrough attribute instead of the freshly-fetched API response.
// The Controller echoes these blocks back with extra server-defaulted fields
// never in the original request (verified live, same pattern as every other
// JSON-passthrough attribute in this provider), which otherwise causes
// Terraform to flag an inconsistent result after Create/Update or show a
// perpetual spurious diff after Read.
func preserveJSONFromPlan(state *webJobResourceModel, src webJobResourceModel) {
	state.ScriptJSON = src.ScriptJSON
	state.ScheduleRunConfigsJSON = src.ScheduleRunConfigsJSON
	state.NetworkProfileJSON = src.NetworkProfileJSON
	state.PerformanceCriteriaJSON = src.PerformanceCriteriaJSON
	state.ComposableConfigJSON = src.ComposableConfigJSON
}

func (r *webJobResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan webJobResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiJob, diags := apiWebJobFromModel(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID := plan.ApplicationID.ValueInt64()
	jobID, err := synthetics.CreateWebJob(ctx, r.client, applicationID, apiJob)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Synthetic Web Job", err.Error())
		return
	}

	created, err := synthetics.GetWebJob(ctx, r.client, applicationID, jobID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Synthetic Web Job After Create", err.Error())
		return
	}

	state, diags := modelFromAPIWebJob(ctx, applicationID, created)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	preserveJSONFromPlan(&state, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *webJobResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state webJobResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID := state.ApplicationID.ValueInt64()
	found, err := synthetics.GetWebJob(ctx, r.client, applicationID, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Synthetic Web Job", err.Error())
		return
	}

	newState, diags := modelFromAPIWebJob(ctx, applicationID, found)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	preserveJSONFromPlan(&newState, state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *webJobResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan webJobResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state webJobResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiJob, diags := apiWebJobFromModel(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	apiJob.ID = state.ID.ValueString()
	apiJob.Version = int(state.Version.ValueInt64())

	applicationID := plan.ApplicationID.ValueInt64()
	if err := synthetics.UpdateWebJob(ctx, r.client, applicationID, apiJob); err != nil {
		resp.Diagnostics.AddError("Error Updating Synthetic Web Job", err.Error())
		return
	}

	updated, err := synthetics.GetWebJob(ctx, r.client, applicationID, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Synthetic Web Job After Update", err.Error())
		return
	}

	newState, diags := modelFromAPIWebJob(ctx, applicationID, updated)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	preserveJSONFromPlan(&newState, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *webJobResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state webJobResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := synthetics.DeleteWebJob(ctx, r.client, state.ApplicationID.ValueInt64(), state.ID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting Synthetic Web Job", err.Error())
	}
}

func (r *webJobResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importCompositeID(ctx, req, resp)
}
