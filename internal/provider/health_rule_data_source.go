package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pmateos/terraform-provider-appdynamics/internal/client"
)

var (
	_ datasource.DataSource              = &healthRuleDataSource{}
	_ datasource.DataSourceWithConfigure = &healthRuleDataSource{}
)

func NewHealthRuleDataSource() datasource.DataSource {
	return &healthRuleDataSource{}
}

type healthRuleDataSource struct {
	client *client.Client
}

type healthRuleDataSourceModel struct {
	ApplicationID           types.Int64          `tfsdk:"application_id"`
	HealthRuleID            types.Int64          `tfsdk:"health_rule_id"`
	Name                    types.String         `tfsdk:"name"`
	Enabled                 types.Bool           `tfsdk:"enabled"`
	UseDataFromLastNMinutes types.Int64          `tfsdk:"use_data_from_last_n_minutes"`
	WaitTimeAfterViolation  types.Int64          `tfsdk:"wait_time_after_violation"`
	ScheduleName            types.String         `tfsdk:"schedule_name"`
	AffectsJSON             jsontypes.Normalized `tfsdk:"affects_json"`
	EvalCriteriasJSON       jsontypes.Normalized `tfsdk:"eval_criterias_json"`
}

func (d *healthRuleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_health_rule"
}

func (d *healthRuleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the details of a specific AppDynamics health rule by ID. Use the appdynamics_health_rules data source to discover health_rule_id values.",
		Attributes: map[string]schema.Attribute{
			"application_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the business application this health rule belongs to.",
			},
			"health_rule_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the health rule to retrieve.",
			},
			"name": schema.StringAttribute{
				Computed: true,
			},
			"enabled": schema.BoolAttribute{
				Computed: true,
			},
			"use_data_from_last_n_minutes": schema.Int64Attribute{
				Computed:    true,
				Description: "Evaluation window in minutes.",
			},
			"wait_time_after_violation": schema.Int64Attribute{
				Computed:    true,
				Description: "Minutes to wait after a violation before re-evaluating.",
			},
			"schedule_name": schema.StringAttribute{
				Computed: true,
			},
			"affects_json": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Computed:    true,
				Description: "JSON object for the Affects block. See the appdynamics_health_rule resource docs for the shape.",
			},
			"eval_criterias_json": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Computed:    true,
				Description: "JSON object for the EvalCriterias block. See the appdynamics_health_rule resource docs for the shape.",
			},
		},
	}
}

func (d *healthRuleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *healthRuleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config healthRuleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := d.client.GetHealthRule(ctx, config.ApplicationID.ValueInt64(), config.HealthRuleID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Health Rule", err.Error())
		return
	}

	state := healthRuleDataSourceModel{
		ApplicationID: config.ApplicationID,
		HealthRuleID:  config.HealthRuleID,
		Name:          types.StringValue(found.Name),
		Enabled:       types.BoolValue(found.Enabled),
		ScheduleName:  stringOrNull(found.ScheduleName),
	}
	if found.UseDataFromLastNMinutes != 0 {
		state.UseDataFromLastNMinutes = types.Int64Value(found.UseDataFromLastNMinutes)
	} else {
		state.UseDataFromLastNMinutes = types.Int64Null()
	}
	if found.WaitTimeAfterViolation != 0 {
		state.WaitTimeAfterViolation = types.Int64Value(found.WaitTimeAfterViolation)
	} else {
		state.WaitTimeAfterViolation = types.Int64Null()
	}
	if len(found.Affects) > 0 {
		state.AffectsJSON = jsontypes.NewNormalizedValue(string(found.Affects))
	} else {
		state.AffectsJSON = jsontypes.NewNormalizedNull()
	}
	if len(found.EvalCriterias) > 0 {
		state.EvalCriteriasJSON = jsontypes.NewNormalizedValue(string(found.EvalCriterias))
	} else {
		state.EvalCriteriasJSON = jsontypes.NewNormalizedNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
