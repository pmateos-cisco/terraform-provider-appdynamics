package rbac

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
	"github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/rbac"
)

var (
	_ resource.Resource                = &roleGroupAssignmentResource{}
	_ resource.ResourceWithConfigure   = &roleGroupAssignmentResource{}
	_ resource.ResourceWithImportState = &roleGroupAssignmentResource{}
)

func NewRoleGroupAssignmentResource() resource.Resource {
	return &roleGroupAssignmentResource{}
}

type roleGroupAssignmentResource struct {
	client *client.Client
}

type roleGroupAssignmentResourceModel struct {
	ID      types.String `tfsdk:"id"`
	RoleID  types.Int64  `tfsdk:"role_id"`
	GroupID types.Int64  `tfsdk:"group_id"`
}

func (r *roleGroupAssignmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_group_assignment"
}

func (r *roleGroupAssignmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Assigns an AppDynamics RBAC role to a group (granting it to every member). Unlike group membership, this IS reflected on the group's own detail (verified live), so this resource's Read queries the group.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"role_id": schema.Int64Attribute{
				Required:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"group_id": schema.Int64Attribute{
				Required:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *roleGroupAssignmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func (r *roleGroupAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan roleGroupAssignmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleID, groupID := plan.RoleID.ValueInt64(), plan.GroupID.ValueInt64()
	if err := rbac.AssignRoleToGroup(ctx, r.client, roleID, groupID); err != nil {
		resp.Diagnostics.AddError("Error Assigning Role to Group", err.Error())
		return
	}

	plan.ID = types.StringValue(membershipID(roleID, groupID))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *roleGroupAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state roleGroupAssignmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := rbac.GetGroup(ctx, r.client, state.GroupID.ValueInt64())
	if err != nil {
		if rbac.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Group for Role Assignment", err.Error())
		return
	}

	roleID := state.RoleID.ValueInt64()
	stillAssigned := false
	for _, ro := range found.Roles {
		if ro.ID == roleID {
			stillAssigned = true
			break
		}
	}
	if !stillAssigned {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update never actually runs: both attributes RequiresReplace. Implemented
// to satisfy the resource.Resource interface.
func (r *roleGroupAssignmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan roleGroupAssignmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *roleGroupAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state roleGroupAssignmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := rbac.RemoveRoleFromGroup(ctx, r.client, state.RoleID.ValueInt64(), state.GroupID.ValueInt64()); err != nil && !rbac.IsNotFound(err) {
		resp.Diagnostics.AddError("Error Removing Role from Group", err.Error())
	}
}

func (r *roleGroupAssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import ID in the form <role_id>/<group_id>, got: %q", req.ID),
		)
		return
	}
	roleID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid role_id in Import Identifier", err.Error())
		return
	}
	groupID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid group_id in Import Identifier", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("role_id"), roleID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_id"), groupID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
