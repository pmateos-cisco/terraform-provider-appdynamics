package database

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
	"github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/database"
)

var (
	_ datasource.DataSource              = &nodesDataSource{}
	_ datasource.DataSourceWithConfigure = &nodesDataSource{}
)

func NewNodesDataSource() datasource.DataSource {
	return &nodesDataSource{}
}

type nodesDataSource struct {
	client *client.Client
}

type nodeSummaryModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	MachineName         types.String `tfsdk:"machine_name"`
	AppAgentVersion     types.String `tfsdk:"app_agent_version"`
	MachineAgentVersion types.String `tfsdk:"machine_agent_version"`
}

type nodesDataSourceModel struct {
	Nodes []nodeSummaryModel `tfsdk:"nodes"`
}

func (d *nodesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_monitoring_nodes"
}

func (d *nodesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the Database Monitoring application nodes (Database Agent instances) in the account (account-wide, no inputs, read-only).",
		Attributes: map[string]schema.Attribute{
			"nodes": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The Database Agent nodes in the account.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Computed: true,
						},
						"machine_name": schema.StringAttribute{
							Computed: true,
						},
						"app_agent_version": schema.StringAttribute{
							Computed: true,
						},
						"machine_agent_version": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *nodesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *nodesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	found, err := database.ListNodes(ctx, d.client)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Database Monitoring Nodes", err.Error())
		return
	}

	nodes := make([]nodeSummaryModel, 0, len(found))
	for _, n := range found {
		nodes = append(nodes, nodeSummaryModel{
			ID:                  types.StringValue(strconv.FormatInt(n.ID, 10)),
			Name:                types.StringValue(n.Name),
			MachineName:         stringOrNull(n.MachineName),
			AppAgentVersion:     stringOrNull(n.AppAgentVersion),
			MachineAgentVersion: stringOrNull(n.MachineAgentVersion),
		})
	}

	state := nodesDataSourceModel{Nodes: nodes}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
