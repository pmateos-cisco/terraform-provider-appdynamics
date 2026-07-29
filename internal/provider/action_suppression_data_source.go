package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client"
)

var (
	_ datasource.DataSource              = &actionSuppressionDataSource{}
	_ datasource.DataSourceWithConfigure = &actionSuppressionDataSource{}
)

func NewActionSuppressionDataSource() datasource.DataSource {
	return &actionSuppressionDataSource{}
}

type actionSuppressionDataSource struct {
	client *client.Client
}

type actionSuppressionDataSourceModel struct {
	ApplicationID           types.Int64          `tfsdk:"application_id"`
	ActionSuppressionID     types.Int64          `tfsdk:"action_suppression_id"`
	Name                    types.String         `tfsdk:"name"`
	DisableAgentReporting   types.Bool           `tfsdk:"disable_agent_reporting"`
	SuppressionScheduleType types.String         `tfsdk:"suppression_schedule_type"`
	Timezone                types.String         `tfsdk:"timezone"`
	StartTime               types.String         `tfsdk:"start_time"`
	EndTime                 types.String         `tfsdk:"end_time"`
	AffectsJSON             jsontypes.Normalized `tfsdk:"affects_json"`
	RecurringScheduleJSON   jsontypes.Normalized `tfsdk:"recurring_schedule_json"`
	HealthRuleScopeJSON     jsontypes.Normalized `tfsdk:"health_rule_scope_json"`
}

func (d *actionSuppressionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_action_suppression"
}

func (d *actionSuppressionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the details of a specific AppDynamics action suppression, looked up by either action_suppression_id or name (exactly one must be set).",
		Attributes: map[string]schema.Attribute{
			"application_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the business application this action suppression belongs to.",
			},
			"action_suppression_id": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "ID of the action suppression to retrieve. Exactly one of action_suppression_id or name must be set.",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Name of the action suppression to retrieve. Exactly one of action_suppression_id or name must be set.",
			},
			"disable_agent_reporting": schema.BoolAttribute{
				Computed: true,
			},
			"suppression_schedule_type": schema.StringAttribute{
				Computed: true,
			},
			"timezone": schema.StringAttribute{
				Computed: true,
			},
			"start_time": schema.StringAttribute{
				Computed: true,
			},
			"end_time": schema.StringAttribute{
				Computed: true,
			},
			"affects_json": schema.StringAttribute{
				CustomType: jsontypes.NormalizedType{},
				Computed:   true,
			},
			"recurring_schedule_json": schema.StringAttribute{
				CustomType: jsontypes.NormalizedType{},
				Computed:   true,
			},
			"health_rule_scope_json": schema.StringAttribute{
				CustomType: jsontypes.NormalizedType{},
				Computed:   true,
			},
		},
	}
}

func (d *actionSuppressionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *actionSuppressionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config actionSuppressionDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasID := !config.ActionSuppressionID.IsNull() && !config.ActionSuppressionID.IsUnknown()
	hasName := !config.Name.IsNull() && !config.Name.IsUnknown()
	if hasID == hasName {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Exactly one of action_suppression_id or name must be set.",
		)
		return
	}

	applicationID := config.ApplicationID.ValueInt64()
	var found *client.ActionSuppression
	var err error
	if hasID {
		found, err = d.client.GetActionSuppression(ctx, applicationID, config.ActionSuppressionID.ValueInt64())
	} else {
		found, err = d.client.GetActionSuppressionByName(ctx, applicationID, config.Name.ValueString())
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Action Suppression", err.Error())
		return
	}

	state := actionSuppressionDataSourceModel{
		ApplicationID:           config.ApplicationID,
		ActionSuppressionID:     types.Int64Value(found.ID),
		Name:                    types.StringValue(found.Name),
		DisableAgentReporting:   types.BoolValue(found.DisableAgentReporting),
		SuppressionScheduleType: types.StringValue(found.SuppressionScheduleType),
		Timezone:                stringOrNull(found.Timezone),
		StartTime:               stringOrNull(found.StartTime),
		EndTime:                 stringOrNull(found.EndTime),
	}
	if len(found.Affects) > 0 {
		state.AffectsJSON = jsontypes.NewNormalizedValue(string(found.Affects))
	} else {
		state.AffectsJSON = jsontypes.NewNormalizedNull()
	}
	if len(found.RecurringSchedule) > 0 && string(found.RecurringSchedule) != "null" {
		state.RecurringScheduleJSON = jsontypes.NewNormalizedValue(string(found.RecurringSchedule))
	} else {
		state.RecurringScheduleJSON = jsontypes.NewNormalizedNull()
	}
	if len(found.HealthRuleScope) > 0 && string(found.HealthRuleScope) != "null" {
		state.HealthRuleScopeJSON = jsontypes.NewNormalizedValue(string(found.HealthRuleScope))
	} else {
		state.HealthRuleScopeJSON = jsontypes.NewNormalizedNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
