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
	_ datasource.DataSource              = &usersDataSource{}
	_ datasource.DataSourceWithConfigure = &usersDataSource{}
)

func NewUsersDataSource() datasource.DataSource {
	return &usersDataSource{}
}

type usersDataSource struct {
	client *client.Client
}

type userSummaryModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

type usersDataSourceModel struct {
	Users []userSummaryModel `tfsdk:"users"`
}

func (d *usersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_users"
}

func (d *usersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the RBAC users in the current AppDynamics account.",
		Attributes: map[string]schema.Attribute{
			"users": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The users in the account.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "User ID.",
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

func (d *usersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *usersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	found, err := rbac.ListUsers(ctx, d.client)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Users", err.Error())
		return
	}

	users := make([]userSummaryModel, 0, len(found))
	for _, u := range found {
		users = append(users, userSummaryModel{
			ID:   types.StringValue(strconv.FormatInt(u.ID, 10)),
			Name: types.StringValue(u.Name),
		})
	}

	state := usersDataSourceModel{Users: users}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
