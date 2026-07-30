package alertandrespond

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
)

var (
	_ datasource.DataSource              = &scheduleDataSource{}
	_ datasource.DataSourceWithConfigure = &scheduleDataSource{}
)

func NewScheduleDataSource() datasource.DataSource {
	return &scheduleDataSource{}
}

type scheduleDataSource struct {
	client *client.Client
}

type scheduleConfigurationDataSourceModel struct {
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

type scheduleDataSourceModel struct {
	ApplicationID         types.Int64                           `tfsdk:"application_id"`
	ScheduleID            types.Int64                           `tfsdk:"schedule_id"`
	Name                  types.String                          `tfsdk:"name"`
	Description           types.String                          `tfsdk:"description"`
	Timezone              types.String                          `tfsdk:"timezone"`
	ScheduleConfiguration *scheduleConfigurationDataSourceModel `tfsdk:"schedule_configuration"`
}

func (d *scheduleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_schedule"
}

func (d *scheduleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the details of a specific AppDynamics alerting schedule by ID. Use the appdynamics_schedules data source to discover schedule_id values.",
		Attributes: map[string]schema.Attribute{
			"application_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the business application this schedule belongs to.",
			},
			"schedule_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the schedule to retrieve.",
			},
			"name": schema.StringAttribute{
				Computed: true,
			},
			"description": schema.StringAttribute{
				Computed: true,
			},
			"timezone": schema.StringAttribute{
				Computed: true,
			},
			"schedule_configuration": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"schedule_frequency": schema.StringAttribute{
						Computed:    true,
						Description: "One of ONE_TIME, DAILY, WEEKLY, MONTHLY_SPECIFIC_DATE, MONTHLY_SPECIFIC_DAY, CUSTOM.",
					},
					"start_date": schema.StringAttribute{
						Computed: true,
					},
					"start_time": schema.StringAttribute{
						Computed: true,
					},
					"end_date": schema.StringAttribute{
						Computed: true,
					},
					"end_time": schema.StringAttribute{
						Computed: true,
					},
					"days": schema.ListAttribute{
						Computed:    true,
						ElementType: types.StringType,
					},
					"day": schema.ListAttribute{
						Computed:    true,
						ElementType: types.StringType,
					},
					"occurrence": schema.StringAttribute{
						Computed: true,
					},
					"start_cron": schema.StringAttribute{
						Computed: true,
					},
					"end_cron": schema.StringAttribute{
						Computed: true,
					},
				},
			},
		},
	}
}

func (d *scheduleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. This is a bug in the provider.", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *scheduleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config scheduleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := d.client.GetSchedule(ctx, config.ApplicationID.ValueInt64(), config.ScheduleID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Schedule", err.Error())
		return
	}

	var diags diag.Diagnostics
	state := scheduleDataSourceModel{
		ApplicationID: config.ApplicationID,
		ScheduleID:    config.ScheduleID,
		Name:          types.StringValue(found.Name),
		Description:   stringOrNull(found.Description),
		Timezone:      types.StringValue(found.Timezone),
	}

	if found.ScheduleConfiguration != nil {
		sc := &scheduleConfigurationDataSourceModel{
			ScheduleFrequency: types.StringValue(found.ScheduleConfiguration.ScheduleFrequency),
			StartDate:         stringOrNull(found.ScheduleConfiguration.StartDate),
			StartTime:         stringOrNull(found.ScheduleConfiguration.StartTime),
			EndDate:           stringOrNull(found.ScheduleConfiguration.EndDate),
			EndTime:           stringOrNull(found.ScheduleConfiguration.EndTime),
			Occurrence:        stringOrNull(found.ScheduleConfiguration.Occurrence),
			StartCron:         stringOrNull(found.ScheduleConfiguration.StartCron),
			EndCron:           stringOrNull(found.ScheduleConfiguration.EndCron),
		}

		days, d := goStringList(ctx, found.ScheduleConfiguration.Days)
		diags.Append(d...)
		sc.Days = days

		day, d := goStringList(ctx, found.ScheduleConfiguration.Day)
		diags.Append(d...)
		sc.Day = day

		state.ScheduleConfiguration = sc
	}

	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
