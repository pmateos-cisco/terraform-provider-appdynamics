package alertandrespond

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
)

var (
	_ datasource.DataSource              = &healthRulesDataSource{}
	_ datasource.DataSourceWithConfigure = &healthRulesDataSource{}
)

func NewHealthRulesDataSource() datasource.DataSource {
	return &healthRulesDataSource{}
}

type healthRulesDataSource struct {
	client *client.Client
}

type healthRuleSummaryModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Enabled            types.Bool   `tfsdk:"enabled"`
	AffectedEntityType types.String `tfsdk:"affected_entity_type"`
}

type healthRulesDataSourceModel struct {
	ApplicationID types.Int64              `tfsdk:"application_id"`
	HealthRules   []healthRuleSummaryModel `tfsdk:"health_rules"`
}

func (d *healthRulesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_health_rules"
}

func (d *healthRulesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the health rules defined for an AppDynamics business application.",
		Attributes: map[string]schema.Attribute{
			"application_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the business application to list health rules for.",
			},
			"health_rules": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The health rules defined for the application.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Health rule ID.",
						},
						"name": schema.StringAttribute{
							Computed: true,
						},
						"enabled": schema.BoolAttribute{
							Computed: true,
						},
						"affected_entity_type": schema.StringAttribute{
							Computed:    true,
							Description: "The entity type this health rule evaluates, e.g. TIER_NODE_HARDWARE, BACKENDS, BUSINESS_TRANSACTION_PERFORMANCE.",
						},
					},
				},
			},
		},
	}
}

func (d *healthRulesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *healthRulesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config healthRulesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := d.client.ListHealthRules(ctx, config.ApplicationID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Health Rules", err.Error())
		return
	}

	rules := make([]healthRuleSummaryModel, 0, len(found))
	for _, hr := range found {
		rules = append(rules, healthRuleSummaryModel{
			ID:                 types.StringValue(strconv.FormatInt(hr.ID, 10)),
			Name:               types.StringValue(hr.Name),
			Enabled:            types.BoolValue(hr.Enabled),
			AffectedEntityType: types.StringValue(hr.AffectedEntityType),
		})
	}

	state := healthRulesDataSourceModel{
		ApplicationID: config.ApplicationID,
		HealthRules:   rules,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
