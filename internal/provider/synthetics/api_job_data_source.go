package synthetics

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
	"github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/synthetics"
)

var (
	_ datasource.DataSource              = &apiJobDataSource{}
	_ datasource.DataSourceWithConfigure = &apiJobDataSource{}
)

func NewAPIJobDataSource() datasource.DataSource {
	return &apiJobDataSource{}
}

type apiJobDataSource struct {
	client *client.Client
}

type apiJobDataSourceModel struct {
	ApplicationID           types.Int64          `tfsdk:"application_id"`
	JobID                   types.String         `tfsdk:"job_id"`
	Description             types.String         `tfsdk:"description"`
	APIMetadataJSON         jsontypes.Normalized `tfsdk:"api_metadata_json"`
	LocationCodes           types.List           `tfsdk:"location_codes"`
	TimeoutSeconds          types.Int64          `tfsdk:"timeout_seconds"`
	UserEnabled             types.Bool           `tfsdk:"user_enabled"`
	ScheduleRunConfigsJSON  jsontypes.Normalized `tfsdk:"schedule_run_configs_json"`
	PerformanceCriteriaJSON jsontypes.Normalized `tfsdk:"performance_criteria_json"`
	ComposableConfigJSON    jsontypes.Normalized `tfsdk:"composable_config_json"`
	Version                 types.Int64          `tfsdk:"version"`
}

func (d *apiJobDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_synthetic_api_job"
}

func (d *apiJobDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the full detail of one Synthetic API Monitoring job by application_id + job_id. Use the appdynamics_synthetic_api_jobs data source to discover job_id values. Shares its type name with the managed resource, same as appdynamics_synthetic_web_job.",
		Attributes: map[string]schema.Attribute{
			"application_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the appdynamics_synthetic_api_application this job is grouped under.",
			},
			"job_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the job to retrieve.",
			},
			"description": schema.StringAttribute{
				Computed: true,
			},
			"api_metadata_json": schema.StringAttribute{
				CustomType: jsontypes.NormalizedType{},
				Computed:   true,
			},
			"location_codes": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"timeout_seconds": schema.Int64Attribute{
				Computed: true,
			},
			"user_enabled": schema.BoolAttribute{
				Computed: true,
			},
			"schedule_run_configs_json": schema.StringAttribute{
				CustomType: jsontypes.NormalizedType{},
				Computed:   true,
			},
			"performance_criteria_json": schema.StringAttribute{
				CustomType: jsontypes.NormalizedType{},
				Computed:   true,
			},
			"composable_config_json": schema.StringAttribute{
				CustomType: jsontypes.NormalizedType{},
				Computed:   true,
			},
			"version": schema.Int64Attribute{
				Computed: true,
			},
		},
	}
}

func (d *apiJobDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *apiJobDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config apiJobDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID := config.ApplicationID.ValueInt64()
	found, err := synthetics.GetAPIJob(ctx, d.client, applicationID, config.JobID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Synthetic API Job", err.Error())
		return
	}

	resourceModel, diags := modelFromAPIAPIJob(ctx, applicationID, found)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := apiJobDataSourceModel{
		ApplicationID:           resourceModel.ApplicationID,
		JobID:                   resourceModel.ID,
		Description:             resourceModel.Description,
		APIMetadataJSON:         resourceModel.APIMetadataJSON,
		LocationCodes:           resourceModel.LocationCodes,
		TimeoutSeconds:          resourceModel.TimeoutSeconds,
		UserEnabled:             resourceModel.UserEnabled,
		ScheduleRunConfigsJSON:  resourceModel.ScheduleRunConfigsJSON,
		PerformanceCriteriaJSON: resourceModel.PerformanceCriteriaJSON,
		ComposableConfigJSON:    resourceModel.ComposableConfigJSON,
		Version:                 resourceModel.Version,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
