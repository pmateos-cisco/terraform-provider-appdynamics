package rbac

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
	"github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/rbac"
)

var (
	_ datasource.DataSource              = &roleDataSource{}
	_ datasource.DataSourceWithConfigure = &roleDataSource{}
)

func NewRoleDataSource() datasource.DataSource {
	return &roleDataSource{}
}

type roleDataSource struct {
	client *client.Client
}

type roleDataSourceModel struct {
	RoleID          types.Int64          `tfsdk:"role_id"`
	Name            types.String         `tfsdk:"name"`
	PermissionsJSON jsontypes.Normalized `tfsdk:"permissions_json"`
}

func (d *roleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (d *roleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the details of a specific AppDynamics RBAC role by ID, including its permissions. Use the appdynamics_roles data source to discover role_id values.",
		Attributes: map[string]schema.Attribute{
			"role_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the role to retrieve.",
			},
			"name": schema.StringAttribute{
				Computed: true,
			},
			"permissions_json": schema.StringAttribute{
				CustomType: jsontypes.NormalizedType{},
				Computed:   true,
			},
		},
	}
}

func (d *roleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *roleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config roleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := rbac.GetRole(ctx, d.client, config.RoleID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Role", err.Error())
		return
	}

	state := roleDataSourceModel{
		RoleID: config.RoleID,
		Name:   types.StringValue(found.Name),
	}
	if len(found.Permissions) > 0 {
		state.PermissionsJSON = jsontypes.NewNormalizedValue(string(found.Permissions))
	} else {
		state.PermissionsJSON = jsontypes.NewNormalizedNull()
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
