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
	_ datasource.DataSource              = &policiesDataSource{}
	_ datasource.DataSourceWithConfigure = &policiesDataSource{}
)

func NewPoliciesDataSource() datasource.DataSource {
	return &policiesDataSource{}
}

type policiesDataSource struct {
	client *client.Client
}

type policySummaryModel struct {
	ID      types.String `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	Enabled types.Bool   `tfsdk:"enabled"`
}

type policiesDataSourceModel struct {
	ApplicationID types.Int64          `tfsdk:"application_id"`
	Policies      []policySummaryModel `tfsdk:"policies"`
}

func (d *policiesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policies"
}

func (d *policiesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the alert policies defined for an AppDynamics business application.",
		Attributes: map[string]schema.Attribute{
			"application_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the business application to list policies for.",
			},
			"policies": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The policies defined for the application.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Policy ID.",
						},
						"name": schema.StringAttribute{
							Computed: true,
						},
						"enabled": schema.BoolAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *policiesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *policiesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config policiesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := d.client.ListPolicies(ctx, config.ApplicationID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Policies", err.Error())
		return
	}

	policies := make([]policySummaryModel, 0, len(found))
	for _, p := range found {
		policies = append(policies, policySummaryModel{
			ID:      types.StringValue(strconv.FormatInt(p.ID, 10)),
			Name:    types.StringValue(p.Name),
			Enabled: types.BoolValue(p.Enabled),
		})
	}

	state := policiesDataSourceModel{
		ApplicationID: config.ApplicationID,
		Policies:      policies,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
