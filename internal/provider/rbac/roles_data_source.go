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
	_ datasource.DataSource              = &rolesDataSource{}
	_ datasource.DataSourceWithConfigure = &rolesDataSource{}
)

func NewRolesDataSource() datasource.DataSource {
	return &rolesDataSource{}
}

type rolesDataSource struct {
	client *client.Client
}

type roleSummaryModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

type rolesDataSourceModel struct {
	Roles []roleSummaryModel `tfsdk:"roles"`
}

func (d *rolesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_roles"
}

func (d *rolesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the RBAC roles in the current AppDynamics account.",
		Attributes: map[string]schema.Attribute{
			"roles": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The roles in the account.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Role ID.",
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

func (d *rolesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *rolesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	found, err := rbac.ListRoles(ctx, d.client)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Roles", err.Error())
		return
	}

	roles := make([]roleSummaryModel, 0, len(found))
	for _, ro := range found {
		roles = append(roles, roleSummaryModel{
			ID:   types.StringValue(strconv.FormatInt(ro.ID, 10)),
			Name: types.StringValue(ro.Name),
		})
	}

	state := rolesDataSourceModel{Roles: roles}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
