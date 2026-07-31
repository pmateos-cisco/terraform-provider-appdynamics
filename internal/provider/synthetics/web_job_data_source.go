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
	_ datasource.DataSource              = &webJobDataSource{}
	_ datasource.DataSourceWithConfigure = &webJobDataSource{}
)

func NewWebJobDataSource() datasource.DataSource {
	return &webJobDataSource{}
}

type webJobDataSource struct {
	client *client.Client
}

type webJobDataSourceModel struct {
	ApplicationID           types.Int64          `tfsdk:"application_id"`
	JobID                   types.String         `tfsdk:"job_id"`
	AppKey                  types.String         `tfsdk:"app_key"`
	Description             types.String         `tfsdk:"description"`
	URL                     types.String         `tfsdk:"url"`
	ScriptJSON              jsontypes.Normalized `tfsdk:"script_json"`
	BrowserCodes            types.List           `tfsdk:"browser_codes"`
	ChromeVersions          types.List           `tfsdk:"chrome_versions"`
	LocationCodes           types.List           `tfsdk:"location_codes"`
	TimeoutSeconds          types.Int64          `tfsdk:"timeout_seconds"`
	UserEnabled             types.Bool           `tfsdk:"user_enabled"`
	ScheduleRunConfigsJSON  jsontypes.Normalized `tfsdk:"schedule_run_configs_json"`
	NetworkProfileJSON      jsontypes.Normalized `tfsdk:"network_profile_json"`
	PerformanceCriteriaJSON jsontypes.Normalized `tfsdk:"performance_criteria_json"`
	ComposableConfigJSON    jsontypes.Normalized `tfsdk:"composable_config_json"`
	Version                 types.Int64          `tfsdk:"version"`
}

func (d *webJobDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_synthetic_web_job"
}

func (d *webJobDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the full detail of one Synthetic Web Monitoring job by application_id + job_id. Use the appdynamics_synthetic_web_jobs data source to discover job_id values. Shares its type name with the managed resource -- resource \"appdynamics_synthetic_web_job\" and data \"appdynamics_synthetic_web_job\" are separate namespaces.",
		Attributes: map[string]schema.Attribute{
			"application_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the business application the Browser RUM app is associated with.",
			},
			"job_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the job to retrieve.",
			},
			"app_key": schema.StringAttribute{
				Computed: true,
			},
			"description": schema.StringAttribute{
				Computed: true,
			},
			"url": schema.StringAttribute{
				Computed: true,
			},
			"script_json": schema.StringAttribute{
				CustomType: jsontypes.NormalizedType{},
				Computed:   true,
			},
			"browser_codes": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"chrome_versions": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
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
			"network_profile_json": schema.StringAttribute{
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

func (d *webJobDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *webJobDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config webJobDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID := config.ApplicationID.ValueInt64()
	found, err := synthetics.GetWebJob(ctx, d.client, applicationID, config.JobID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Synthetic Web Job", err.Error())
		return
	}

	resourceModel, diags := modelFromAPIWebJob(ctx, applicationID, found)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := webJobDataSourceModel{
		ApplicationID:           resourceModel.ApplicationID,
		JobID:                   resourceModel.ID,
		AppKey:                  resourceModel.AppKey,
		Description:             resourceModel.Description,
		URL:                     resourceModel.URL,
		ScriptJSON:              resourceModel.ScriptJSON,
		BrowserCodes:            resourceModel.BrowserCodes,
		ChromeVersions:          resourceModel.ChromeVersions,
		LocationCodes:           resourceModel.LocationCodes,
		TimeoutSeconds:          resourceModel.TimeoutSeconds,
		UserEnabled:             resourceModel.UserEnabled,
		ScheduleRunConfigsJSON:  resourceModel.ScheduleRunConfigsJSON,
		NetworkProfileJSON:      resourceModel.NetworkProfileJSON,
		PerformanceCriteriaJSON: resourceModel.PerformanceCriteriaJSON,
		ComposableConfigJSON:    resourceModel.ComposableConfigJSON,
		Version:                 resourceModel.Version,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
