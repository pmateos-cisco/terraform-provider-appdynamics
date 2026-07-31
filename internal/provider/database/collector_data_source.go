package database

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
	"github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/database"
)

var (
	_ datasource.DataSource              = &collectorDataSource{}
	_ datasource.DataSourceWithConfigure = &collectorDataSource{}
)

func NewCollectorDataSource() datasource.DataSource {
	return &collectorDataSource{}
}

type collectorDataSource struct {
	client *client.Client
}

type collectorDataSourceModel struct {
	CollectorID     types.String         `tfsdk:"collector_id"`
	Type            types.String         `tfsdk:"type"`
	Name            types.String         `tfsdk:"name"`
	Hostname        types.String         `tfsdk:"hostname"`
	Port            types.Int64          `tfsdk:"port"`
	Username        types.String         `tfsdk:"username"`
	AgentName       types.String         `tfsdk:"agent_name"`
	Enabled         types.Bool           `tfsdk:"enabled"`
	ExtraConfigJSON jsontypes.Normalized `tfsdk:"extra_config_json"`
	Version         types.Int64          `tfsdk:"version"`
}

func (d *collectorDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_collector"
}

func (d *collectorDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the full detail of one Database Visibility collector by ID (password excluded -- the API never returns it). Shares its type name with the managed resource -- resource \"appdynamics_database_collector\" and data \"appdynamics_database_collector\" are separate namespaces.",
		Attributes: map[string]schema.Attribute{
			"collector_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the collector to retrieve.",
			},
			"type": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Computed: true,
			},
			"hostname": schema.StringAttribute{
				Computed: true,
			},
			"port": schema.Int64Attribute{
				Computed: true,
			},
			"username": schema.StringAttribute{
				Computed: true,
			},
			"agent_name": schema.StringAttribute{
				Computed: true,
			},
			"enabled": schema.BoolAttribute{
				Computed: true,
			},
			"extra_config_json": schema.StringAttribute{
				CustomType: jsontypes.NormalizedType{},
				Computed:   true,
			},
			"version": schema.Int64Attribute{
				Computed: true,
			},
		},
	}
}

func (d *collectorDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *collectorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config collectorDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	collectorID, err := strconv.ParseInt(config.CollectorID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid collector_id", err.Error())
		return
	}

	found, err := database.GetCollector(ctx, d.client, collectorID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Database Collector", err.Error())
		return
	}

	resourceModel := modelFromAPICollector(found)
	state := collectorDataSourceModel{
		CollectorID:     resourceModel.ID,
		Type:            resourceModel.Type,
		Name:            resourceModel.Name,
		Hostname:        resourceModel.Hostname,
		Port:            resourceModel.Port,
		Username:        resourceModel.Username,
		AgentName:       resourceModel.AgentName,
		Enabled:         resourceModel.Enabled,
		ExtraConfigJSON: resourceModel.ExtraConfigJSON,
		Version:         resourceModel.Version,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
