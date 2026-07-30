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
	_ datasource.DataSource              = &actionSuppressionsDataSource{}
	_ datasource.DataSourceWithConfigure = &actionSuppressionsDataSource{}
)

func NewActionSuppressionsDataSource() datasource.DataSource {
	return &actionSuppressionsDataSource{}
}

type actionSuppressionsDataSource struct {
	client *client.Client
}

type actionSuppressionSummaryModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Timezone  types.String `tfsdk:"timezone"`
	StartTime types.String `tfsdk:"start_time"`
	EndTime   types.String `tfsdk:"end_time"`
}

type actionSuppressionsDataSourceModel struct {
	ApplicationID      types.Int64                     `tfsdk:"application_id"`
	ActionSuppressions []actionSuppressionSummaryModel `tfsdk:"action_suppressions"`
}

func (d *actionSuppressionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_action_suppressions"
}

func (d *actionSuppressionsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the action suppressions defined for an AppDynamics business application.",
		Attributes: map[string]schema.Attribute{
			"application_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the business application to list action suppressions for.",
			},
			"action_suppressions": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The action suppressions defined for the application.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Action suppression ID.",
						},
						"name": schema.StringAttribute{
							Computed: true,
						},
						"timezone": schema.StringAttribute{
							Computed: true,
						},
						"start_time": schema.StringAttribute{
							Computed: true,
						},
						"end_time": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *actionSuppressionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *actionSuppressionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config actionSuppressionsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := d.client.ListActionSuppressions(ctx, config.ApplicationID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Action Suppressions", err.Error())
		return
	}

	suppressions := make([]actionSuppressionSummaryModel, 0, len(found))
	for _, as := range found {
		suppressions = append(suppressions, actionSuppressionSummaryModel{
			ID:        types.StringValue(strconv.FormatInt(as.ID, 10)),
			Name:      types.StringValue(as.Name),
			Timezone:  stringOrNull(as.Timezone),
			StartTime: stringOrNull(as.StartTime),
			EndTime:   stringOrNull(as.EndTime),
		})
	}

	state := actionSuppressionsDataSourceModel{
		ApplicationID:      config.ApplicationID,
		ActionSuppressions: suppressions,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
