package synthetics

import (
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
	_ resource.Resource                = &apiJobResource{}
	_ resource.ResourceWithConfigure   = &apiJobResource{}
	_ resource.ResourceWithImportState = &apiJobResource{}
)

func NewAPIJobResource() resource.Resource {
	return &apiJobResource{}
}

type apiJobResource struct {
	client *client.Client
}

type apiJobResourceModel struct {
	ID                      types.String         `tfsdk:"id"`
	ApplicationID           types.Int64          `tfsdk:"application_id"`
	Description             types.String         `tfsdk:"description"`
	APIMetadataJSON         jsontypes.Normalized `tfsdk:"api_metadata_json"`
	LocationCodes           types.List           `tfsdk:"location_codes"`
	TimeoutSeconds          types.Int64          `tfsdk:"timeout_seconds"`
	UserEnabled             types.Bool           `tfsdk:"user_enabled"`
	ScheduleRunConfigsJSON  jsontypes.Normalized `tfsdk:"schedule_run_configs_json"`
	PerformanceCriteriaJSON jsontypes.Normalized `tfsdk:"performance_criteria_json"`
	ComposableConfigJSON    jsontypes.Normalized `tfsdk:"composable_config_json"`
	Version                 types.Int64          `tfsdk:"version"`
}

func (r *apiJobResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_synthetic_api_job"
}

func (r *apiJobResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An AppDynamics Synthetic API Monitoring job: a scheduled scripted check of an API endpoint, grouped under an appdynamics_synthetic_api_application. Uses the Controller's own internal API (verified live), same as appdynamics_synthetic_web_job.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"application_id": schema.Int64Attribute{
				Required:      true,
				Description:   "ID of the appdynamics_synthetic_api_application this job is grouped under.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"description": schema.StringAttribute{
				Required:    true,
				Description: "Job name, shown in the Controller UI's Jobs list.",
			},
			"api_metadata_json": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Required:    true,
				Description: "JSON object {\"script\":{\"contentType\":\"JAVASCRIPT\",\"script\":\"...\"}} describing the API check script.",
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
				Description: "Whether the job actively runs on its schedule. Defaults to false, same as appdynamics_synthetic_web_job (verified live).",
			},
			"schedule_run_configs_json": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Required:    true,
				Description: "JSON array of run-schedule configs, e.g. [{\"rate\":{\"value\":15,\"unit\":\"MINUTES\"},\"daysOfWeek\":[\"MON\"],\"timezone\":\"UTC\"}]. rate.value must be 1-59 for MINUTES.",
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

func (r *apiJobResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func apiAPIJobFromModel(ctx context.Context, m apiJobResourceModel) (*synthetics.APIJob, diag.Diagnostics) {
	var diags diag.Diagnostics

	locationCodes, d := stringListToGo(ctx, m.LocationCodes)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	job := &synthetics.APIJob{
		Description:        m.Description.ValueString(),
		APIMetadata:        json.RawMessage(m.APIMetadataJSON.ValueString()),
		LocationCodes:      locationCodes,
		TimeoutSeconds:     int(m.TimeoutSeconds.ValueInt64()),
		UserEnabled:        m.UserEnabled.ValueBool(),
		ScheduleRunConfigs: json.RawMessage(m.ScheduleRunConfigsJSON.ValueString()),
	}
	if !m.PerformanceCriteriaJSON.IsNull() && !m.PerformanceCriteriaJSON.IsUnknown() {
		job.PerformanceCriteria = json.RawMessage(m.PerformanceCriteriaJSON.ValueString())
	}
	if !m.ComposableConfigJSON.IsNull() && !m.ComposableConfigJSON.IsUnknown() {
		job.ComposableConfig = json.RawMessage(m.ComposableConfigJSON.ValueString())
	}
	return job, diags
}

func modelFromAPIAPIJob(ctx context.Context, applicationID int64, job *synthetics.APIJob) (apiJobResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	locationCodes, d := goStringList(ctx, job.LocationCodes)
	diags.Append(d...)

	m := apiJobResourceModel{
		ID:             types.StringValue(job.ID),
		ApplicationID:  types.Int64Value(applicationID),
		Description:    types.StringValue(job.Description),
		LocationCodes:  locationCodes,
		TimeoutSeconds: types.Int64Value(int64(job.TimeoutSeconds)),
		UserEnabled:    types.BoolValue(job.UserEnabled),
		Version:        types.Int64Value(int64(job.Version)),
	}
	if hasJSON(job.APIMetadata) {
		m.APIMetadataJSON = jsontypes.NewNormalizedValue(string(job.APIMetadata))
	} else {
		m.APIMetadataJSON = jsontypes.NewNormalizedNull()
	}
	if hasJSON(job.ScheduleRunConfigs) {
		m.ScheduleRunConfigsJSON = jsontypes.NewNormalizedValue(string(job.ScheduleRunConfigs))
	} else {
		m.ScheduleRunConfigsJSON = jsontypes.NewNormalizedNull()
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

// preserveAPIJobJSONFromPlan keeps the plan's/prior state's own value for
// every JSON-passthrough attribute, same rationale as
// preserveJSONFromPlan for appdynamics_synthetic_web_job.
func preserveAPIJobJSONFromPlan(state *apiJobResourceModel, src apiJobResourceModel) {
	state.APIMetadataJSON = src.APIMetadataJSON
	state.ScheduleRunConfigsJSON = src.ScheduleRunConfigsJSON
	state.PerformanceCriteriaJSON = src.PerformanceCriteriaJSON
	state.ComposableConfigJSON = src.ComposableConfigJSON
}

func (r *apiJobResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan apiJobResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiJob, diags := apiAPIJobFromModel(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID := plan.ApplicationID.ValueInt64()
	jobID, err := synthetics.CreateAPIJob(ctx, r.client, applicationID, apiJob)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Synthetic API Job", err.Error())
		return
	}

	created, err := synthetics.GetAPIJob(ctx, r.client, applicationID, jobID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Synthetic API Job After Create", err.Error())
		return
	}

	state, diags := modelFromAPIAPIJob(ctx, applicationID, created)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	preserveAPIJobJSONFromPlan(&state, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *apiJobResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state apiJobResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID := state.ApplicationID.ValueInt64()
	found, err := synthetics.GetAPIJob(ctx, r.client, applicationID, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Synthetic API Job", err.Error())
		return
	}

	newState, diags := modelFromAPIAPIJob(ctx, applicationID, found)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	preserveAPIJobJSONFromPlan(&newState, state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *apiJobResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan apiJobResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state apiJobResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiJob, diags := apiAPIJobFromModel(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	apiJob.ID = state.ID.ValueString()
	apiJob.Version = int(state.Version.ValueInt64())

	applicationID := plan.ApplicationID.ValueInt64()
	if err := synthetics.UpdateAPIJob(ctx, r.client, applicationID, apiJob); err != nil {
		resp.Diagnostics.AddError("Error Updating Synthetic API Job", err.Error())
		return
	}

	updated, err := synthetics.GetAPIJob(ctx, r.client, applicationID, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Synthetic API Job After Update", err.Error())
		return
	}

	newState, diags := modelFromAPIAPIJob(ctx, applicationID, updated)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	preserveAPIJobJSONFromPlan(&newState, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *apiJobResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state apiJobResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := synthetics.DeleteAPIJob(ctx, r.client, state.ApplicationID.ValueInt64(), state.ID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting Synthetic API Job", err.Error())
	}
}

func (r *apiJobResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importCompositeID(ctx, req, resp)
}
