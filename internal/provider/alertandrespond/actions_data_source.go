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
	_ datasource.DataSource              = &actionsDataSource{}
	_ datasource.DataSourceWithConfigure = &actionsDataSource{}
)

func NewActionsDataSource() datasource.DataSource {
	return &actionsDataSource{}
}

type actionsDataSource struct {
	client *client.Client
}

type actionSummaryModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	ActionType types.String `tfsdk:"action_type"`
}

type actionsDataSourceModel struct {
	ApplicationID types.Int64          `tfsdk:"application_id"`
	Actions       []actionSummaryModel `tfsdk:"actions"`
}

func (d *actionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_actions"
}

func (d *actionsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the alert actions defined for an AppDynamics business application.",
		Attributes: map[string]schema.Attribute{
			"application_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the business application to list actions for.",
			},
			"actions": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The actions defined for the application.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Action ID.",
						},
						"name": schema.StringAttribute{
							Computed: true,
						},
						"action_type": schema.StringAttribute{
							Computed:    true,
							Description: "One of SMS, EMAIL, CUSTOM_EMAIL, THREAD_DUMP, HTTP_REQUEST, RUN_SCRIPT_ON_NODES, DIAGNOSE_BUSINESS_TRANSACTIONS, CREATE_UPDATE_JIRA.",
						},
					},
				},
			},
		},
	}
}

func (d *actionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *actionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config actionsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := d.client.ListActions(ctx, config.ApplicationID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Actions", err.Error())
		return
	}

	actions := make([]actionSummaryModel, 0, len(found))
	for _, a := range found {
		actions = append(actions, actionSummaryModel{
			ID:         types.StringValue(strconv.FormatInt(a.ID, 10)),
			Name:       types.StringValue(a.Name),
			ActionType: types.StringValue(a.ActionType),
		})
	}

	state := actionsDataSourceModel{
		ApplicationID: config.ApplicationID,
		Actions:       actions,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
