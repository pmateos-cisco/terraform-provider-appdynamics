package alertandrespond

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
)

var (
	_ datasource.DataSource              = &eventsDataSource{}
	_ datasource.DataSourceWithConfigure = &eventsDataSource{}
)

func NewEventsDataSource() datasource.DataSource {
	return &eventsDataSource{}
}

type eventsDataSource struct {
	client *client.Client
}

type eventSummaryModel struct {
	ID          types.String `tfsdk:"id"`
	Type        types.String `tfsdk:"type"`
	SubType     types.String `tfsdk:"sub_type"`
	Severity    types.String `tfsdk:"severity"`
	Summary     types.String `tfsdk:"summary"`
	EventTime   types.Int64  `tfsdk:"event_time"`
	DeepLinkURL types.String `tfsdk:"deep_link_url"`
}

type eventsDataSourceModel struct {
	ApplicationID  types.Int64         `tfsdk:"application_id"`
	TimeRangeType  types.String        `tfsdk:"time_range_type"`
	DurationInMins types.Int64         `tfsdk:"duration_in_mins"`
	StartTime      types.Int64         `tfsdk:"start_time"`
	EndTime        types.Int64         `tfsdk:"end_time"`
	EventTypes     types.List          `tfsdk:"event_types"`
	Severities     types.List          `tfsdk:"severities"`
	Events         []eventSummaryModel `tfsdk:"events"`
}

func (d *eventsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_events"
}

func (d *eventsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Queries AppDynamics events for an application within a time range. This is a " +
			"reporting/lookup query over historical data, not a list of currently-managed config objects " +
			"-- results change over time as new events occur, and re-running the same query later can " +
			"return different results even with identical arguments.",
		Attributes: map[string]schema.Attribute{
			"application_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the business application to query events for.",
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
			"event_types": schema.ListAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "Event types to include, e.g. [\"CUSTOM\", \"APPLICATION_DEPLOYMENT\"].",
			},
			"severities": schema.ListAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "Severities to include, e.g. [\"INFO\", \"WARN\", \"ERROR\"].",
			},
			"events": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The events matching the query.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
						"type": schema.StringAttribute{
							Computed: true,
						},
						"sub_type": schema.StringAttribute{
							Computed: true,
						},
						"severity": schema.StringAttribute{
							Computed: true,
						},
						"summary": schema.StringAttribute{
							Computed: true,
						},
						"event_time": schema.Int64Attribute{
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

func (d *eventsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *eventsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config eventsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	eventTypes, diags := stringListToGo(ctx, config.EventTypes)
	resp.Diagnostics.Append(diags...)
	severities, diags := stringListToGo(ctx, config.Severities)
	resp.Diagnostics.Append(diags...)
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
	params.Set("event-types", strings.Join(eventTypes, ","))
	params.Set("severities", strings.Join(severities, ","))

	found, err := d.client.ListEvents(ctx, config.ApplicationID.ValueInt64(), params)
	if err != nil {
		resp.Diagnostics.AddError("Error Querying Events", err.Error())
		return
	}

	events := make([]eventSummaryModel, 0, len(found))
	for _, e := range found {
		events = append(events, eventSummaryModel{
			ID:          types.StringValue(strconv.FormatInt(e.ID, 10)),
			Type:        types.StringValue(e.Type),
			SubType:     stringOrNull(e.SubType),
			Severity:    types.StringValue(e.Severity),
			Summary:     types.StringValue(e.Summary),
			EventTime:   types.Int64Value(e.EventTimeMillis),
			DeepLinkURL: stringOrNull(e.DeepLinkURL),
		})
	}

	state := config
	state.Events = events
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
