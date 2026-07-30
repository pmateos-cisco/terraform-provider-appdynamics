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
	_ resource.Resource                = &roleUserAssignmentResource{}
	_ resource.ResourceWithConfigure   = &roleUserAssignmentResource{}
	_ resource.ResourceWithImportState = &roleUserAssignmentResource{}
)

func NewRoleUserAssignmentResource() resource.Resource {
	return &roleUserAssignmentResource{}
}

type roleUserAssignmentResource struct {
	client *client.Client
}

type roleUserAssignmentResourceModel struct {
	ID     types.String `tfsdk:"id"`
	RoleID types.Int64  `tfsdk:"role_id"`
	UserID types.Int64  `tfsdk:"user_id"`
}

func (r *roleUserAssignmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_user_assignment"
}

func (r *roleUserAssignmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Assigns an AppDynamics RBAC role directly to a user. Reflected on the user's implicit roles list (verified live).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"role_id": schema.Int64Attribute{
				Required:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"user_id": schema.Int64Attribute{
				Required:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *roleUserAssignmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func (r *roleUserAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan roleUserAssignmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleID, userID := plan.RoleID.ValueInt64(), plan.UserID.ValueInt64()
	if err := rbac.AssignRoleToUser(ctx, r.client, roleID, userID); err != nil {
		resp.Diagnostics.AddError("Error Assigning Role to User", err.Error())
		return
	}

	plan.ID = types.StringValue(membershipID(roleID, userID))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *roleUserAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state roleUserAssignmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := rbac.GetUser(ctx, r.client, state.UserID.ValueInt64())
	if err != nil {
		if rbac.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading User for Role Assignment", err.Error())
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
func (r *roleUserAssignmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan roleUserAssignmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *roleUserAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state roleUserAssignmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := rbac.RemoveRoleFromUser(ctx, r.client, state.RoleID.ValueInt64(), state.UserID.ValueInt64()); err != nil && !rbac.IsNotFound(err) {
		resp.Diagnostics.AddError("Error Removing Role from User", err.Error())
	}
}

func (r *roleUserAssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import ID in the form <role_id>/<user_id>, got: %q", req.ID),
		)
		return
	}
	roleID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid role_id in Import Identifier", err.Error())
		return
	}
	userID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid user_id in Import Identifier", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("role_id"), roleID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), userID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
