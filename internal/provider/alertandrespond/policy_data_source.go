package alertandrespond

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
)

var (
	_ datasource.DataSource              = &policyDataSource{}
	_ datasource.DataSourceWithConfigure = &policyDataSource{}
)

func NewPolicyDataSource() datasource.DataSource {
	return &policyDataSource{}
}

type policyDataSource struct {
	client *client.Client
}

type policyDataSourceModel struct {
	ApplicationID         types.Int64          `tfsdk:"application_id"`
	PolicyID              types.Int64          `tfsdk:"policy_id"`
	Name                  types.String         `tfsdk:"name"`
	Enabled               types.Bool           `tfsdk:"enabled"`
	ExecuteActionsInBatch types.Bool           `tfsdk:"execute_actions_in_batch"`
	ActionsJSON           jsontypes.Normalized `tfsdk:"actions_json"`
	EventsJSON            jsontypes.Normalized `tfsdk:"events_json"`
	SelectedEntitiesJSON  jsontypes.Normalized `tfsdk:"selected_entities_json"`
}

func (d *policyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

func (d *policyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the details of a specific AppDynamics policy by ID. Use the appdynamics_policies data source to discover policy_id values.",
		Attributes: map[string]schema.Attribute{
			"application_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the business application this policy belongs to.",
			},
			"policy_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the policy to retrieve.",
			},
			"name": schema.StringAttribute{
				Computed: true,
			},
			"enabled": schema.BoolAttribute{
				Computed: true,
			},
			"execute_actions_in_batch": schema.BoolAttribute{
				Computed: true,
			},
			"actions_json": schema.StringAttribute{
				CustomType: jsontypes.NormalizedType{},
				Computed:   true,
			},
			"events_json": schema.StringAttribute{
				CustomType: jsontypes.NormalizedType{},
				Computed:   true,
			},
			"selected_entities_json": schema.StringAttribute{
				CustomType: jsontypes.NormalizedType{},
				Computed:   true,
			},
		},
	}
}

func (d *policyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *policyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config policyDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := d.client.GetPolicy(ctx, config.ApplicationID.ValueInt64(), config.PolicyID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Policy", err.Error())
		return
	}

	state := policyDataSourceModel{
		ApplicationID:         config.ApplicationID,
		PolicyID:              config.PolicyID,
		Name:                  types.StringValue(found.Name),
		Enabled:               types.BoolValue(found.Enabled),
		ExecuteActionsInBatch: types.BoolValue(found.ExecuteActionsInBatch),
	}
	if len(found.Actions) > 0 {
		state.ActionsJSON = jsontypes.NewNormalizedValue(string(found.Actions))
	} else {
		state.ActionsJSON = jsontypes.NewNormalizedNull()
	}
	if len(found.Events) > 0 {
		state.EventsJSON = jsontypes.NewNormalizedValue(string(found.Events))
	} else {
		state.EventsJSON = jsontypes.NewNormalizedNull()
	}
	if len(found.SelectedEntities) > 0 {
		state.SelectedEntitiesJSON = jsontypes.NewNormalizedValue(string(found.SelectedEntities))
	} else {
		state.SelectedEntitiesJSON = jsontypes.NewNormalizedNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
