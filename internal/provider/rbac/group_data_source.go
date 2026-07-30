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
	_ datasource.DataSource              = &groupDataSource{}
	_ datasource.DataSourceWithConfigure = &groupDataSource{}
)

func NewGroupDataSource() datasource.DataSource {
	return &groupDataSource{}
}

type groupDataSource struct {
	client *client.Client
}

type groupDataSourceModel struct {
	GroupID              types.Int64  `tfsdk:"group_id"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	SecurityProviderType types.String `tfsdk:"security_provider_type"`
}

func (d *groupDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (d *groupDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the details of a specific AppDynamics RBAC group, looked up by either group_id or name (exactly one must be set).",
		Attributes: map[string]schema.Attribute{
			"group_id": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "ID of the group to retrieve. Exactly one of group_id or name must be set.",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Name of the group to retrieve. Exactly one of group_id or name must be set.",
			},
			"description": schema.StringAttribute{
				Computed: true,
			},
			"security_provider_type": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *groupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *groupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config groupDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasID := !config.GroupID.IsNull() && !config.GroupID.IsUnknown()
	hasName := !config.Name.IsNull() && !config.Name.IsUnknown()
	if hasID == hasName {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Exactly one of group_id or name must be set.",
		)
		return
	}

	var found *rbac.Group
	var err error
	if hasID {
		found, err = rbac.GetGroup(ctx, d.client, config.GroupID.ValueInt64())
	} else {
		found, err = rbac.GetGroupByName(ctx, d.client, config.Name.ValueString())
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Group", err.Error())
		return
	}

	state := groupDataSourceModel{
		GroupID:              types.Int64Value(found.ID),
		Name:                 types.StringValue(found.Name),
		Description:          stringOrNull(found.Description),
		SecurityProviderType: types.StringValue(found.SecurityProviderType),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
