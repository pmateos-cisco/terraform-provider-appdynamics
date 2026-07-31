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
	_ datasource.DataSource              = &apiApplicationsDataSource{}
	_ datasource.DataSourceWithConfigure = &apiApplicationsDataSource{}
)

func NewAPIApplicationsDataSource() datasource.DataSource {
	return &apiApplicationsDataSource{}
}

type apiApplicationsDataSource struct {
	client *client.Client
}

type apiApplicationSummaryModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

type apiApplicationsDataSourceModel struct {
	Applications []apiApplicationSummaryModel `tfsdk:"applications"`
}

func (d *apiApplicationsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_synthetic_api_applications"
}

func (d *apiApplicationsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all Synthetic API Monitoring applications in the account (account-wide, no inputs).",
		Attributes: map[string]schema.Attribute{
			"applications": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The Synthetic API Monitoring applications in the account (id, name only; use the singular appdynamics_synthetic_api_application data source for app_key).",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *apiApplicationsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *apiApplicationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	found, err := synthetics.ListAPIApplications(ctx, d.client)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Synthetic API Applications", err.Error())
		return
	}

	apps := make([]apiApplicationSummaryModel, 0, len(found))
	for _, a := range found {
		apps = append(apps, apiApplicationSummaryModel{
			ID:   types.StringValue(strconv.FormatInt(a.ID, 10)),
			Name: types.StringValue(a.Name),
		})
	}

	state := apiApplicationsDataSourceModel{Applications: apps}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
