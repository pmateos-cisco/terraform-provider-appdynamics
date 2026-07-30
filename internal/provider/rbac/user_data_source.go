package rbac

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
	"github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/rbac"
)

var (
	_ datasource.DataSource              = &userDataSource{}
	_ datasource.DataSourceWithConfigure = &userDataSource{}
)

func NewUserDataSource() datasource.DataSource {
	return &userDataSource{}
}

type userDataSource struct {
	client *client.Client
}

type userDataSourceModel struct {
	UserID               types.Int64  `tfsdk:"user_id"`
	Name                 types.String `tfsdk:"name"`
	DisplayName          types.String `tfsdk:"display_name"`
	SecurityProviderType types.String `tfsdk:"security_provider_type"`
}

func (d *userDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *userDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the details of a specific AppDynamics RBAC user, looked up by either user_id or name (exactly one must be set).",
		Attributes: map[string]schema.Attribute{
			"user_id": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "ID of the user to retrieve. Exactly one of user_id or name must be set.",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Login username of the user to retrieve. Exactly one of user_id or name must be set.",
			},
			"display_name": schema.StringAttribute{
				Computed: true,
			},
			"security_provider_type": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *userDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *userDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config userDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasID := !config.UserID.IsNull() && !config.UserID.IsUnknown()
	hasName := !config.Name.IsNull() && !config.Name.IsUnknown()
	if hasID == hasName {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Exactly one of user_id or name must be set.",
		)
		return
	}

	var found *rbac.User
	var err error
	if hasID {
		found, err = rbac.GetUser(ctx, d.client, config.UserID.ValueInt64())
	} else {
		found, err = rbac.GetUserByName(ctx, d.client, config.Name.ValueString())
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading User", err.Error())
		return
	}

	state := userDataSourceModel{
		UserID:               types.Int64Value(found.ID),
		Name:                 types.StringValue(found.Name),
		DisplayName:          stringOrNull(found.DisplayName),
		SecurityProviderType: types.StringValue(found.SecurityProviderType),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
