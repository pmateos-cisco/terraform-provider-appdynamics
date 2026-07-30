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
	_ resource.Resource                = &groupMembershipResource{}
	_ resource.ResourceWithConfigure   = &groupMembershipResource{}
	_ resource.ResourceWithImportState = &groupMembershipResource{}
)

func NewGroupMembershipResource() resource.Resource {
	return &groupMembershipResource{}
}

type groupMembershipResource struct {
	client *client.Client
}

type groupMembershipResourceModel struct {
	ID      types.String `tfsdk:"id"`
	GroupID types.Int64  `tfsdk:"group_id"`
	UserID  types.Int64  `tfsdk:"user_id"`
}

func (r *groupMembershipResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_membership"
}

func (r *groupMembershipResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Adds a user to an AppDynamics RBAC group. Membership is only ever reflected on the user (via appdynamics_user's implicit groups), not on the group itself (verified live), so this resource's Read queries the user.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"group_id": schema.Int64Attribute{
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

func (r *groupMembershipResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func membershipID(groupID, userID int64) string {
	return fmt.Sprintf("%d/%d", groupID, userID)
}

func (r *groupMembershipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan groupMembershipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID, userID := plan.GroupID.ValueInt64(), plan.UserID.ValueInt64()
	if err := rbac.AddUserToGroup(ctx, r.client, groupID, userID); err != nil {
		resp.Diagnostics.AddError("Error Adding User to Group", err.Error())
		return
	}

	plan.ID = types.StringValue(membershipID(groupID, userID))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *groupMembershipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state groupMembershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userID := state.UserID.ValueInt64()
	found, err := rbac.GetUser(ctx, r.client, userID)
	if err != nil {
		if rbac.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading User for Group Membership", err.Error())
		return
	}

	groupID := state.GroupID.ValueInt64()
	stillMember := false
	for _, g := range found.Groups {
		if g.ID == groupID {
			stillMember = true
			break
		}
	}
	if !stillMember {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update never actually runs: both attributes RequiresReplace. Implemented
// to satisfy the resource.Resource interface.
func (r *groupMembershipResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan groupMembershipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *groupMembershipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state groupMembershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := rbac.RemoveUserFromGroup(ctx, r.client, state.GroupID.ValueInt64(), state.UserID.ValueInt64()); err != nil && !rbac.IsNotFound(err) {
		resp.Diagnostics.AddError("Error Removing User from Group", err.Error())
	}
}

func (r *groupMembershipResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import ID in the form <group_id>/<user_id>, got: %q", req.ID),
		)
		return
	}
	groupID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid group_id in Import Identifier", err.Error())
		return
	}
	userID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid user_id in Import Identifier", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_id"), groupID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), userID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
