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
	_ datasource.DataSource              = &webJobsDataSource{}
	_ datasource.DataSourceWithConfigure = &webJobsDataSource{}
)

func NewWebJobsDataSource() datasource.DataSource {
	return &webJobsDataSource{}
}

type webJobsDataSource struct {
	client *client.Client
}

type webJobSummaryModel struct {
	ID          types.String `tfsdk:"id"`
	Description types.String `tfsdk:"description"`
	URL         types.String `tfsdk:"url"`
	UserEnabled types.Bool   `tfsdk:"user_enabled"`
}

type webJobsDataSourceModel struct {
	ApplicationID types.Int64          `tfsdk:"application_id"`
	Jobs          []webJobSummaryModel `tfsdk:"jobs"`
}

func (d *webJobsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_synthetic_web_jobs"
}

func (d *webJobsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the Synthetic Web Monitoring jobs configured for a business application's Browser RUM app.",
		Attributes: map[string]schema.Attribute{
			"application_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the business application to list jobs for.",
			},
			"jobs": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The jobs configured for the application (id, description, url, user_enabled only; use the singular appdynamics_synthetic_web_job data source for full detail on one job).",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
						"description": schema.StringAttribute{
							Computed: true,
						},
						"url": schema.StringAttribute{
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

func (d *webJobsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *webJobsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config webJobsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID := config.ApplicationID.ValueInt64()
	found, err := synthetics.ListWebJobs(ctx, d.client, applicationID)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Synthetic Web Jobs", err.Error())
		return
	}

	jobs := make([]webJobSummaryModel, 0, len(found))
	for _, j := range found {
		jobs = append(jobs, webJobSummaryModel{
			ID:          types.StringValue(j.ID),
			Description: types.StringValue(j.Description),
			URL:         stringOrNull(j.URL),
			UserEnabled: types.BoolValue(j.UserEnabled),
		})
	}

	state := webJobsDataSourceModel{
		ApplicationID: config.ApplicationID,
		Jobs:          jobs,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
