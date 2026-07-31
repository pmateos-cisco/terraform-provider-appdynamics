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
	_ datasource.DataSource              = &collectorsDataSource{}
	_ datasource.DataSourceWithConfigure = &collectorsDataSource{}
)

func NewCollectorsDataSource() datasource.DataSource {
	return &collectorsDataSource{}
}

type collectorsDataSource struct {
	client *client.Client
}

type collectorSummaryModel struct {
	ID       types.String `tfsdk:"id"`
	Type     types.String `tfsdk:"type"`
	Name     types.String `tfsdk:"name"`
	Hostname types.String `tfsdk:"hostname"`
	Enabled  types.Bool   `tfsdk:"enabled"`
}

type collectorsDataSourceModel struct {
	Collectors []collectorSummaryModel `tfsdk:"collectors"`
}

func (d *collectorsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_collectors"
}

func (d *collectorsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all Database Visibility collectors in the account (account-wide, no inputs).",
		Attributes: map[string]schema.Attribute{
			"collectors": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The collectors in the account (id, type, name, hostname, enabled only; use the singular appdynamics_database_collector data source for full detail on one collector).",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
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
						"enabled": schema.BoolAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *collectorsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *collectorsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	found, err := database.ListCollectors(ctx, d.client)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Database Collectors", err.Error())
		return
	}

	collectors := make([]collectorSummaryModel, 0, len(found))
	for _, c := range found {
		collectors = append(collectors, collectorSummaryModel{
			ID:       types.StringValue(strconv.FormatInt(c.ID, 10)),
			Type:     types.StringValue(c.Type),
			Name:     types.StringValue(c.Name),
			Hostname: types.StringValue(c.Hostname),
			Enabled:  types.BoolValue(c.Enabled),
		})
	}

	state := collectorsDataSourceModel{Collectors: collectors}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
