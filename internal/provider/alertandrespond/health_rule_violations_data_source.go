package alertandrespond

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
)

var (
	_ datasource.DataSource              = &healthRuleViolationsDataSource{}
	_ datasource.DataSourceWithConfigure = &healthRuleViolationsDataSource{}
)

func NewHealthRuleViolationsDataSource() datasource.DataSource {
	return &healthRuleViolationsDataSource{}
}

type healthRuleViolationsDataSource struct {
	client *client.Client
}

type violationSummaryModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Severity       types.String `tfsdk:"severity"`
	IncidentStatus types.String `tfsdk:"incident_status"`
	Description    types.String `tfsdk:"description"`
	StartTime      types.Int64  `tfsdk:"start_time"`
	EndTime        types.Int64  `tfsdk:"end_time"`
	DeepLinkURL    types.String `tfsdk:"deep_link_url"`
}

type healthRuleViolationsDataSourceModel struct {
	ApplicationID  types.Int64             `tfsdk:"application_id"`
	TimeRangeType  types.String            `tfsdk:"time_range_type"`
	DurationInMins types.Int64             `tfsdk:"duration_in_mins"`
	StartTime      types.Int64             `tfsdk:"start_time"`
	EndTime        types.Int64             `tfsdk:"end_time"`
	Violations     []violationSummaryModel `tfsdk:"violations"`
}

func (d *healthRuleViolationsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_health_rule_violations"
}

func (d *healthRuleViolationsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Queries AppDynamics health rule violations (incidents) for an application within " +
			"a time range. This is a reporting/lookup query over historical data, not a list of " +
			"currently-managed config objects -- results change over time as new violations occur, and " +
			"re-running the same query later can return different results even with identical arguments.",
		Attributes: map[string]schema.Attribute{
			"application_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the business application to query violations for.",
			},
			"time_range_type": schema.StringAttribute{
				Required:    true,
				Description: "BEFORE_NOW, AFTER_TIME, or BETWEEN_TIMES. See the AppDynamics Events API docs.",
			},
			"duration_in_mins": schema.Int64Attribute{
				Optional:    true,
				Description: "Required when time_range_type is BEFORE_NOW or AFTER_TIME.",
			},
			"start_time": schema.Int64Attribute{
				Optional:    true,
				Description: "Epoch milliseconds. Required when time_range_type is BETWEEN_TIMES or AFTER_TIME.",
			},
			"end_time": schema.Int64Attribute{
				Optional:    true,
				Description: "Epoch milliseconds. Required when time_range_type is BETWEEN_TIMES.",
			},
			"violations": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The health rule violations matching the query.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Computed: true,
						},
						"severity": schema.StringAttribute{
							Computed: true,
						},
						"incident_status": schema.StringAttribute{
							Computed: true,
						},
						"description": schema.StringAttribute{
							Computed: true,
						},
						"start_time": schema.Int64Attribute{
							Computed:    true,
							Description: "Epoch milliseconds.",
						},
						"end_time": schema.Int64Attribute{
							Computed:    true,
							Description: "Epoch milliseconds.",
						},
						"deep_link_url": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *healthRuleViolationsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *healthRuleViolationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config healthRuleViolationsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := url.Values{}
	params.Set("time-range-type", config.TimeRangeType.ValueString())
	if !config.DurationInMins.IsNull() {
		params.Set("duration-in-mins", strconv.FormatInt(config.DurationInMins.ValueInt64(), 10))
	}
	if !config.StartTime.IsNull() {
		params.Set("start-time", strconv.FormatInt(config.StartTime.ValueInt64(), 10))
	}
	if !config.EndTime.IsNull() {
		params.Set("end-time", strconv.FormatInt(config.EndTime.ValueInt64(), 10))
	}

	found, err := d.client.ListHealthRuleViolations(ctx, config.ApplicationID.ValueInt64(), params)
	if err != nil {
		resp.Diagnostics.AddError("Error Querying Health Rule Violations", err.Error())
		return
	}

	violations := make([]violationSummaryModel, 0, len(found))
	for _, v := range found {
		violations = append(violations, violationSummaryModel{
			ID:             types.StringValue(strconv.FormatInt(v.ID, 10)),
			Name:           types.StringValue(v.Name),
			Severity:       types.StringValue(v.Severity),
			IncidentStatus: types.StringValue(v.IncidentStatus),
			Description:    stringOrNull(v.Description),
			StartTime:      types.Int64Value(v.StartTimeInMillis),
			EndTime:        types.Int64Value(v.EndTimeInMillis),
			DeepLinkURL:    stringOrNull(v.DeepLinkURL),
		})
	}

	state := config
	state.Violations = violations
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
