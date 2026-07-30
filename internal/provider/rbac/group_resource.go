package rbac

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
	"github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/rbac"
)

var (
	_ resource.Resource                = &groupResource{}
	_ resource.ResourceWithConfigure   = &groupResource{}
	_ resource.ResourceWithImportState = &groupResource{}
)

func NewGroupResource() resource.Resource {
	return &groupResource{}
}

type groupResource struct {
	client *client.Client
}

type groupResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	SecurityProviderType types.String `tfsdk:"security_provider_type"`
}

func (r *groupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (r *groupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An AppDynamics RBAC group, account-wide (not scoped to a business application). Use appdynamics_group_membership to add/remove users.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"description": schema.StringAttribute{
				Optional: true,
			},
			"security_provider_type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("INTERNAL"),
			},
		},
	}
}

func (r *groupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func apiGroupFromModel(m groupResourceModel) *rbac.Group {
	return &rbac.Group{
		Name:                 m.Name.ValueString(),
		Description:          m.Description.ValueString(),
		SecurityProviderType: m.SecurityProviderType.ValueString(),
	}
}

func modelFromAPIGroup(g *rbac.Group) groupResourceModel {
	return groupResourceModel{
		ID:                   types.StringValue(strconv.FormatInt(g.ID, 10)),
		Name:                 types.StringValue(g.Name),
		Description:          stringOrNull(g.Description),
		SecurityProviderType: types.StringValue(g.SecurityProviderType),
	}
}

func (r *groupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan groupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := rbac.CreateGroup(ctx, r.client, apiGroupFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Group", err.Error())
		return
	}

	state := modelFromAPIGroup(created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *groupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state groupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Group ID in State", err.Error())
		return
	}

	found, err := rbac.GetGroup(ctx, r.client, groupID)
	if err != nil {
		if rbac.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Group", err.Error())
		return
	}

	newState := modelFromAPIGroup(found)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *groupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan groupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state groupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Group ID in State", err.Error())
		return
	}

	apiGroup := apiGroupFromModel(plan)
	apiGroup.ID = groupID
	updated, err := rbac.UpdateGroup(ctx, r.client, apiGroup)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Group", err.Error())
		return
	}

	newState := modelFromAPIGroup(updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *groupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state groupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Group ID in State", err.Error())
		return
	}

	if err := rbac.DeleteGroup(ctx, r.client, groupID); err != nil && !rbac.IsNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting Group", err.Error())
	}
}

func (r *groupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
