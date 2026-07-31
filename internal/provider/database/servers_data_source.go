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
	_ datasource.DataSource              = &serversDataSource{}
	_ datasource.DataSourceWithConfigure = &serversDataSource{}
)

func NewServersDataSource() datasource.DataSource {
	return &serversDataSource{}
}

type serversDataSource struct {
	client *client.Client
}

type serverSummaryModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Type types.String `tfsdk:"type"`
	Host types.String `tfsdk:"host"`
}

type serversDataSourceModel struct {
	Servers []serverSummaryModel `tfsdk:"servers"`
}

func (d *serversDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_servers"
}

func (d *serversDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all monitored database servers discovered by Database Visibility collectors (account-wide, no inputs, read-only -- there is no create/update/delete API for these).",
		Attributes: map[string]schema.Attribute{
			"servers": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The monitored database servers in the account (id, name, type, host only; use the singular appdynamics_database_server data source for full detail on one server).",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Computed: true,
						},
						"type": schema.StringAttribute{
							Computed: true,
						},
						"host": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *serversDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *serversDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	found, err := database.ListServers(ctx, d.client)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Database Servers", err.Error())
		return
	}

	servers := make([]serverSummaryModel, 0, len(found))
	for _, s := range found {
		servers = append(servers, serverSummaryModel{
			ID:   types.StringValue(strconv.FormatInt(s.ID, 10)),
			Name: types.StringValue(s.Name),
			Type: types.StringValue(s.Type),
			Host: types.StringValue(s.Host),
		})
	}

	state := serversDataSourceModel{Servers: servers}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
