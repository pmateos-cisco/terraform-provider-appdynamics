package rbac

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
	"github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/rbac"
)

var (
	_ resource.Resource                = &roleResource{}
	_ resource.ResourceWithConfigure   = &roleResource{}
	_ resource.ResourceWithImportState = &roleResource{}
)

func NewRoleResource() resource.Resource {
	return &roleResource{}
}

type roleResource struct {
	client *client.Client
}

type roleResourceModel struct {
	ID              types.String         `tfsdk:"id"`
	Name            types.String         `tfsdk:"name"`
	PermissionsJSON jsontypes.Normalized `tfsdk:"permissions_json"`
}

func (r *roleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *roleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An AppDynamics RBAC role: a named set of permissions, account-wide (not scoped to a business application). Use appdynamics_role_user_assignment / appdynamics_role_group_assignment to assign it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"permissions_json": schema.StringAttribute{
				CustomType:    jsontypes.NormalizedType{},
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "JSON array of permissions, e.g. [{\"entityType\":\"APPLICATION\",\"action\":\"VIEW\"}]. See the Splunk AppDynamics RBAC API docs for the shape. Permissions cannot be changed on an existing role (verified live: PUT rejects any request containing a permissions field with \"Users are not allowed to create/update permissions\"), so changing this forces replacement of the role.",
			},
		},
	}
}

func (r *roleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func apiRoleFromModel(m roleResourceModel) *rbac.Role {
	role := &rbac.Role{
		Name: m.Name.ValueString(),
	}
	if !m.PermissionsJSON.IsNull() && !m.PermissionsJSON.IsUnknown() {
		role.Permissions = json.RawMessage(m.PermissionsJSON.ValueString())
	}
	return role
}

func modelFromAPIRole(role *rbac.Role) roleResourceModel {
	m := roleResourceModel{
		ID:   types.StringValue(strconv.FormatInt(role.ID, 10)),
		Name: types.StringValue(role.Name),
	}
	if len(role.Permissions) > 0 {
		m.PermissionsJSON = jsontypes.NewNormalizedValue(string(role.Permissions))
	} else {
		m.PermissionsJSON = jsontypes.NewNormalizedNull()
	}
	return m
}

func (r *roleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan roleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := rbac.CreateRole(ctx, r.client, apiRoleFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Role", err.Error())
		return
	}

	// The create response only ever contains id/name (verified live) -- read
	// the role back with include-permissions=true to get a consistent
	// starting point before applying the same passthrough-preservation fix
	// used everywhere else in this provider.
	full, err := rbac.GetRole(ctx, r.client, created.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Role After Create", err.Error())
		return
	}

	state := modelFromAPIRole(full)
	// permissions_json is Required (not Computed): the API echoes permissions
	// back with extra server-assigned fields (id, entityId, tagList) the plan
	// never specified, so state must keep the plan's own value or Terraform
	// flags an inconsistent result.
	state.PermissionsJSON = plan.PermissionsJSON
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *roleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state roleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Role ID in State", err.Error())
		return
	}

	found, err := rbac.GetRole(ctx, r.client, roleID)
	if err != nil {
		if rbac.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Role", err.Error())
		return
	}

	newState := modelFromAPIRole(found)
	// See the comment in Create: keep the prior state's value instead of the
	// freshly-fetched one to avoid a perpetual, spurious diff on every plan.
	newState.PermissionsJSON = state.PermissionsJSON
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *roleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan roleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state roleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Role ID in State", err.Error())
		return
	}

	apiRole := apiRoleFromModel(plan)
	apiRole.ID = roleID
	// The API rejects any PUT that includes a permissions field at all, even
	// resending the unchanged value (verified live), so it must never be
	// sent on update -- permissions_json is RequiresReplace, so Update only
	// ever runs here for a name change.
	apiRole.Permissions = nil
	if _, err := rbac.UpdateRole(ctx, r.client, apiRole); err != nil {
		resp.Diagnostics.AddError("Error Updating Role", err.Error())
		return
	}

	full, err := rbac.GetRole(ctx, r.client, roleID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Role After Update", err.Error())
		return
	}

	newState := modelFromAPIRole(full)
	newState.PermissionsJSON = plan.PermissionsJSON
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *roleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state roleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Role ID in State", err.Error())
		return
	}

	if err := rbac.DeleteRole(ctx, r.client, roleID); err != nil && !rbac.IsNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting Role", err.Error())
	}
}

func (r *roleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
