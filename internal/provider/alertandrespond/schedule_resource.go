package alertandrespond

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
)

var (
	_ resource.Resource                = &scheduleResource{}
	_ resource.ResourceWithConfigure   = &scheduleResource{}
	_ resource.ResourceWithImportState = &scheduleResource{}
)

func NewScheduleResource() resource.Resource {
	return &scheduleResource{}
}

type scheduleResource struct {
	client *client.Client
}

type scheduleResourceModel struct {
	ID                    types.String                `tfsdk:"id"`
	ApplicationID         types.Int64                 `tfsdk:"application_id"`
	Name                  types.String                `tfsdk:"name"`
	Description           types.String                `tfsdk:"description"`
	Timezone              types.String                `tfsdk:"timezone"`
	ScheduleConfiguration *scheduleConfigurationModel `tfsdk:"schedule_configuration"`
}

type scheduleConfigurationModel struct {
	ScheduleFrequency types.String `tfsdk:"schedule_frequency"`
	StartDate         types.String `tfsdk:"start_date"`
	StartTime         types.String `tfsdk:"start_time"`
	EndDate           types.String `tfsdk:"end_date"`
	EndTime           types.String `tfsdk:"end_time"`
	Days              types.List   `tfsdk:"days"`
	Day               types.List   `tfsdk:"day"`
	Occurrence        types.String `tfsdk:"occurrence"`
	StartCron         types.String `tfsdk:"start_cron"`
	EndCron           types.String `tfsdk:"end_cron"`
}

func (r *scheduleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_schedule"
}

func (r *scheduleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An AppDynamics alerting schedule, used to control when health rules are evaluated.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"application_id": schema.Int64Attribute{
				Required:      true,
				Description:   "ID of the business application this schedule belongs to.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"description": schema.StringAttribute{
				Optional: true,
			},
			"timezone": schema.StringAttribute{
				Required:    true,
				Description: "IANA timezone ID, e.g. America/Los_Angeles.",
			},
			"schedule_configuration": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"schedule_frequency": schema.StringAttribute{
						Required:    true,
						Description: "One of ONE_TIME, DAILY, WEEKLY, MONTHLY_SPECIFIC_DATE, MONTHLY_SPECIFIC_DAY, CUSTOM.",
					},
					"start_date": schema.StringAttribute{
						Optional:    true,
						Description: "DD/MM/YYYY. Used by ONE_TIME and MONTHLY_SPECIFIC_DATE.",
					},
					"start_time": schema.StringAttribute{
						Optional:    true,
						Description: "HH:MM (24-hour). Used by all frequencies except CUSTOM.",
					},
					"end_date": schema.StringAttribute{
						Optional:    true,
						Description: "DD/MM/YYYY. Used by ONE_TIME and MONTHLY_SPECIFIC_DATE.",
					},
					"end_time": schema.StringAttribute{
						Optional:    true,
						Description: "HH:MM (24-hour). Used by all frequencies except CUSTOM.",
					},
					"days": schema.ListAttribute{
						Optional:    true,
						ElementType: types.StringType,
						Description: "Weekday names (e.g. MONDAY). Used by WEEKLY.",
					},
					"day": schema.ListAttribute{
						Optional:    true,
						ElementType: types.StringType,
						Description: "Weekday names. Used by MONTHLY_SPECIFIC_DAY.",
					},
					"occurrence": schema.StringAttribute{
						Optional:    true,
						Description: "One of FIRST, SECOND, THIRD, FOURTH, LAST. Used by MONTHLY_SPECIFIC_DAY.",
					},
					"start_cron": schema.StringAttribute{
						Optional:    true,
						Description: "UNIX cron expression. Used by CUSTOM.",
					},
					"end_cron": schema.StringAttribute{
						Optional:    true,
						Description: "UNIX cron expression. Used by CUSTOM.",
					},
				},
			},
		},
	}
}

func (r *scheduleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func (r *scheduleResource) apiScheduleFromModel(ctx context.Context, m scheduleResourceModel) (*client.Schedule, diag.Diagnostics) {
	var diags diag.Diagnostics

	sc := &client.ScheduleConfiguration{
		ScheduleFrequency: m.ScheduleConfiguration.ScheduleFrequency.ValueString(),
		StartDate:         m.ScheduleConfiguration.StartDate.ValueString(),
		StartTime:         m.ScheduleConfiguration.StartTime.ValueString(),
		EndDate:           m.ScheduleConfiguration.EndDate.ValueString(),
		EndTime:           m.ScheduleConfiguration.EndTime.ValueString(),
		Occurrence:        m.ScheduleConfiguration.Occurrence.ValueString(),
		StartCron:         m.ScheduleConfiguration.StartCron.ValueString(),
		EndCron:           m.ScheduleConfiguration.EndCron.ValueString(),
	}

	days, d := stringListToGo(ctx, m.ScheduleConfiguration.Days)
	diags.Append(d...)
	sc.Days = days

	day, d := stringListToGo(ctx, m.ScheduleConfiguration.Day)
	diags.Append(d...)
	sc.Day = day

	return &client.Schedule{
		Name:                  m.Name.ValueString(),
		Description:           m.Description.ValueString(),
		Timezone:              m.Timezone.ValueString(),
		ScheduleConfiguration: sc,
	}, diags
}

func (r *scheduleResource) modelFromAPISchedule(ctx context.Context, applicationID int64, s *client.Schedule) (scheduleResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	m := scheduleResourceModel{
		ID:            types.StringValue(strconv.FormatInt(s.ID, 10)),
		ApplicationID: types.Int64Value(applicationID),
		Name:          types.StringValue(s.Name),
		Timezone:      types.StringValue(s.Timezone),
	}
	if s.Description != "" {
		m.Description = types.StringValue(s.Description)
	} else {
		m.Description = types.StringNull()
	}

	if s.ScheduleConfiguration != nil {
		sc := &scheduleConfigurationModel{
			ScheduleFrequency: types.StringValue(s.ScheduleConfiguration.ScheduleFrequency),
			StartDate:         stringOrNull(s.ScheduleConfiguration.StartDate),
			StartTime:         stringOrNull(s.ScheduleConfiguration.StartTime),
			EndDate:           stringOrNull(s.ScheduleConfiguration.EndDate),
			EndTime:           stringOrNull(s.ScheduleConfiguration.EndTime),
			Occurrence:        stringOrNull(s.ScheduleConfiguration.Occurrence),
			StartCron:         stringOrNull(s.ScheduleConfiguration.StartCron),
			EndCron:           stringOrNull(s.ScheduleConfiguration.EndCron),
		}

		days, d := goStringList(ctx, s.ScheduleConfiguration.Days)
		diags.Append(d...)
		sc.Days = days

		day, d := goStringList(ctx, s.ScheduleConfiguration.Day)
		diags.Append(d...)
		sc.Day = day

		m.ScheduleConfiguration = sc
	}

	return m, diags
}

func (r *scheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan scheduleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiSchedule, diags := r.apiScheduleFromModel(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateSchedule(ctx, plan.ApplicationID.ValueInt64(), apiSchedule)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Schedule", err.Error())
		return
	}

	state, diags := r.modelFromAPISchedule(ctx, plan.ApplicationID.ValueInt64(), created)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *scheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state scheduleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scheduleID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Schedule ID in State", err.Error())
		return
	}

	found, err := r.client.GetSchedule(ctx, state.ApplicationID.ValueInt64(), scheduleID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Schedule", err.Error())
		return
	}

	newState, diags := r.modelFromAPISchedule(ctx, state.ApplicationID.ValueInt64(), found)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *scheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan scheduleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state scheduleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scheduleID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Schedule ID in State", err.Error())
		return
	}

	apiSchedule, diags := r.apiScheduleFromModel(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateSchedule(ctx, plan.ApplicationID.ValueInt64(), scheduleID, apiSchedule)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Schedule", err.Error())
		return
	}

	newState, diags := r.modelFromAPISchedule(ctx, plan.ApplicationID.ValueInt64(), updated)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *scheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state scheduleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scheduleID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Schedule ID in State", err.Error())
		return
	}

	if err := r.client.DeleteSchedule(ctx, state.ApplicationID.ValueInt64(), scheduleID); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting Schedule", err.Error())
	}
}

func (r *scheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importCompositeID(ctx, req, resp)
}

func stringOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}
