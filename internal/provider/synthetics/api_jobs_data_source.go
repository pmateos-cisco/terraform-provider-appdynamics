package synthetics

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
	"github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/synthetics"
)

var (
	_ datasource.DataSource              = &apiJobsDataSource{}
	_ datasource.DataSourceWithConfigure = &apiJobsDataSource{}
)

func NewAPIJobsDataSource() datasource.DataSource {
	return &apiJobsDataSource{}
}

type apiJobsDataSource struct {
	client *client.Client
}

type apiJobSummaryModel struct {
	ID          types.String `tfsdk:"id"`
	Description types.String `tfsdk:"description"`
	UserEnabled types.Bool   `tfsdk:"user_enabled"`
}

type apiJobsDataSourceModel struct {
	ApplicationID types.Int64          `tfsdk:"application_id"`
	Jobs          []apiJobSummaryModel `tfsdk:"jobs"`
}

func (d *apiJobsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_synthetic_api_jobs"
}

func (d *apiJobsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the Synthetic API Monitoring jobs configured under a Synthetic API Monitoring application.",
		Attributes: map[string]schema.Attribute{
			"application_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the appdynamics_synthetic_api_application to list jobs for.",
			},
			"jobs": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The jobs configured for the application (id, description, user_enabled only; use the singular appdynamics_synthetic_api_job data source for full detail on one job).",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
						"description": schema.StringAttribute{
							Computed: true,
						},
						"user_enabled": schema.BoolAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *apiJobsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *apiJobsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config apiJobsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID := config.ApplicationID.ValueInt64()
	found, err := synthetics.ListAPIJobs(ctx, d.client, applicationID)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Synthetic API Jobs", err.Error())
		return
	}

	jobs := make([]apiJobSummaryModel, 0, len(found))
	for _, j := range found {
		jobs = append(jobs, apiJobSummaryModel{
			ID:          types.StringValue(j.ID),
			Description: types.StringValue(j.Description),
			UserEnabled: types.BoolValue(j.UserEnabled),
		})
	}

	state := apiJobsDataSourceModel{
		ApplicationID: config.ApplicationID,
		Jobs:          jobs,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
