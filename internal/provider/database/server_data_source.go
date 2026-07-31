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
	_ datasource.DataSource              = &serverDataSource{}
	_ datasource.DataSourceWithConfigure = &serverDataSource{}
)

func NewServerDataSource() datasource.DataSource {
	return &serverDataSource{}
}

type serverDataSource struct {
	client *client.Client
}

type serverDataSourceModel struct {
	ServerID     types.String `tfsdk:"server_id"`
	Name         types.String `tfsdk:"name"`
	Type         types.String `tfsdk:"type"`
	Role         types.String `tfsdk:"role"`
	Host         types.String `tfsdk:"host"`
	Port         types.Int64  `tfsdk:"port"`
	IPAddress    types.String `tfsdk:"ip_address"`
	NodeID       types.String `tfsdk:"node_id"`
	ConfigID     types.String `tfsdk:"config_id"`
	InternalName types.String `tfsdk:"internal_name"`
}

func (d *serverDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_server"
}

func (d *serverDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the detail of one monitored database server by ID. Read-only.",
		Attributes: map[string]schema.Attribute{
			"server_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the server to retrieve.",
			},
			"name": schema.StringAttribute{
				Computed: true,
			},
			"type": schema.StringAttribute{
				Computed: true,
			},
			"role": schema.StringAttribute{
				Computed: true,
			},
			"host": schema.StringAttribute{
				Computed: true,
			},
			"port": schema.Int64Attribute{
				Computed: true,
			},
			"ip_address": schema.StringAttribute{
				Computed: true,
			},
			"node_id": schema.StringAttribute{
				Computed: true,
			},
			"config_id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the appdynamics_database_collector that discovered this server.",
			},
			"internal_name": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *serverDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *serverDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config serverDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverID, err := strconv.ParseInt(config.ServerID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid server_id", err.Error())
		return
	}

	found, err := database.GetServer(ctx, d.client, serverID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Database Server", err.Error())
		return
	}

	state := serverDataSourceModel{
		ServerID:     types.StringValue(strconv.FormatInt(found.ID, 10)),
		Name:         types.StringValue(found.Name),
		Type:         types.StringValue(found.Type),
		Role:         stringOrNull(found.Role),
		Host:         types.StringValue(found.Host),
		Port:         types.Int64Value(int64(found.Port)),
		IPAddress:    stringOrNull(found.IPAddress),
		NodeID:       types.StringValue(strconv.FormatInt(found.NodeID, 10)),
		ConfigID:     types.StringValue(strconv.FormatInt(found.ConfigID, 10)),
		InternalName: stringOrNull(found.InternalName),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
