package alertandrespond

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
)

var (
	_ datasource.DataSource              = &schedulesDataSource{}
	_ datasource.DataSourceWithConfigure = &schedulesDataSource{}
)

func NewSchedulesDataSource() datasource.DataSource {
	return &schedulesDataSource{}
}

type schedulesDataSource struct {
	client *client.Client
}

type scheduleSummaryModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Timezone    types.String `tfsdk:"timezone"`
}

type schedulesDataSourceModel struct {
	ApplicationID types.Int64            `tfsdk:"application_id"`
	Schedules     []scheduleSummaryModel `tfsdk:"schedules"`
}

func (d *schedulesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_schedules"
}

func (d *schedulesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the alerting schedules defined for an AppDynamics business application.",
		Attributes: map[string]schema.Attribute{
			"application_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the business application to list schedules for.",
			},
			"schedules": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The schedules defined for the application.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Schedule ID.",
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
					},
				},
			},
		},
	}
}

func (d *schedulesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *schedulesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config schedulesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := d.client.ListSchedules(ctx, config.ApplicationID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Schedules", err.Error())
		return
	}

	schedules := make([]scheduleSummaryModel, 0, len(found))
	for _, s := range found {
		schedules = append(schedules, scheduleSummaryModel{
			ID:          types.StringValue(strconv.FormatInt(s.ID, 10)),
			Name:        types.StringValue(s.Name),
			Description: stringOrNull(s.Description),
			Timezone:    types.StringValue(s.Timezone),
		})
	}

	state := schedulesDataSourceModel{
		ApplicationID: config.ApplicationID,
		Schedules:     schedules,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
