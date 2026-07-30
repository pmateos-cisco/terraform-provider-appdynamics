package rbac

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
	"github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/rbac"
)

var (
	_ datasource.DataSource              = &groupsDataSource{}
	_ datasource.DataSourceWithConfigure = &groupsDataSource{}
)

func NewGroupsDataSource() datasource.DataSource {
	return &groupsDataSource{}
}

type groupsDataSource struct {
	client *client.Client
}

type groupSummaryModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

type groupsDataSourceModel struct {
	Groups []groupSummaryModel `tfsdk:"groups"`
}

func (d *groupsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_groups"
}

func (d *groupsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the RBAC groups in the current AppDynamics account.",
		Attributes: map[string]schema.Attribute{
			"groups": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The groups in the account.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Group ID.",
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

func (d *groupsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *groupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	found, err := rbac.ListGroups(ctx, d.client)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Groups", err.Error())
		return
	}

	groups := make([]groupSummaryModel, 0, len(found))
	for _, g := range found {
		groups = append(groups, groupSummaryModel{
			ID:   types.StringValue(strconv.FormatInt(g.ID, 10)),
			Name: types.StringValue(g.Name),
		})
	}

	state := groupsDataSourceModel{Groups: groups}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
