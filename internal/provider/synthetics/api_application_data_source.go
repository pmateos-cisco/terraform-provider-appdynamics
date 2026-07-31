package synthetics

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
	"github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/synthetics"
)

var (
	_ datasource.DataSource              = &apiApplicationDataSource{}
	_ datasource.DataSourceWithConfigure = &apiApplicationDataSource{}
)

func NewAPIApplicationDataSource() datasource.DataSource {
	return &apiApplicationDataSource{}
}

type apiApplicationDataSource struct {
	client *client.Client
}

type apiApplicationDataSourceModel struct {
	ApplicationID types.String `tfsdk:"application_id"`
	Name          types.String `tfsdk:"name"`
	AppKey        types.String `tfsdk:"app_key"`
}

func (d *apiApplicationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_synthetic_api_application"
}

func (d *apiApplicationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the detail (including app_key) of one Synthetic API Monitoring application by ID. Shares its type name with the managed resource -- resource \"appdynamics_synthetic_api_application\" and data \"appdynamics_synthetic_api_application\" are separate namespaces.",
		Attributes: map[string]schema.Attribute{
			"application_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the application to retrieve.",
			},
			"name": schema.StringAttribute{
				Computed: true,
			},
			"app_key": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *apiApplicationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *apiApplicationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config apiApplicationDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID, err := strconv.ParseInt(config.ApplicationID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid application_id", err.Error())
		return
	}

	found, err := synthetics.GetAPIApplication(ctx, d.client, applicationID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Synthetic API Application", err.Error())
		return
	}

	state := apiApplicationDataSourceModel{
		ApplicationID: config.ApplicationID,
		Name:          types.StringValue(found.Name),
		AppKey:        stringOrNull(found.AppKey),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
