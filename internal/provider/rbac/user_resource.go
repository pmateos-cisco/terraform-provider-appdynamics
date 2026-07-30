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
	_ resource.Resource                = &userResource{}
	_ resource.ResourceWithConfigure   = &userResource{}
	_ resource.ResourceWithImportState = &userResource{}
)

func NewUserResource() resource.Resource {
	return &userResource{}
}

type userResource struct {
	client *client.Client
}

type userResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	DisplayName          types.String `tfsdk:"display_name"`
	SecurityProviderType types.String `tfsdk:"security_provider_type"`
	Password             types.String `tfsdk:"password"`
}

func (r *userResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *userResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An AppDynamics RBAC user account, account-wide (not scoped to a business application).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Login username. Mutable in place (verified live) -- changing it updates the existing user rather than replacing it.",
			},
			"display_name": schema.StringAttribute{
				Optional: true,
			},
			"security_provider_type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("INTERNAL"),
			},
			"password": schema.StringAttribute{
				Optional:      true,
				Sensitive:     true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Write-only: the API never returns the password in any response, so this value cannot be verified or refreshed from the Controller. There is no password-update endpoint (verified live: PUT rejects the request with \"'password' should not specify\" even when resending the unchanged value), so changing this forces replacement of the user.",
			},
		},
	}
}

func (r *userResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func apiUserFromModel(m userResourceModel) *rbac.User {
	return &rbac.User{
		Name:                 m.Name.ValueString(),
		DisplayName:          m.DisplayName.ValueString(),
		SecurityProviderType: m.SecurityProviderType.ValueString(),
		Password:             m.Password.ValueString(),
	}
}

func modelFromAPIUser(u *rbac.User) userResourceModel {
	return userResourceModel{
		ID:                   types.StringValue(strconv.FormatInt(u.ID, 10)),
		Name:                 types.StringValue(u.Name),
		DisplayName:          stringOrNull(u.DisplayName),
		SecurityProviderType: types.StringValue(u.SecurityProviderType),
	}
}

func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := rbac.CreateUser(ctx, r.client, apiUserFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error Creating User", err.Error())
		return
	}

	state := modelFromAPIUser(created)
	// password is Optional (not Computed) but the API never echoes it back;
	// keep the plan's own value or Terraform flags an inconsistent result.
	state.Password = plan.Password
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid User ID in State", err.Error())
		return
	}

	found, err := rbac.GetUser(ctx, r.client, userID)
	if err != nil {
		if rbac.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading User", err.Error())
		return
	}

	newState := modelFromAPIUser(found)
	// See the comment in Create: password never comes from the API, so keep
	// whatever was already in state.
	newState.Password = state.Password
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan userResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid User ID in State", err.Error())
		return
	}

	apiUser := apiUserFromModel(plan)
	apiUser.ID = userID
	// The API rejects any PUT that includes a password field at all, even
	// resending the unchanged value (verified live), so it must never be
	// sent on update -- password is create-only (see the schema's
	// RequiresReplace on this field).
	apiUser.Password = ""
	updated, err := rbac.UpdateUser(ctx, r.client, apiUser)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating User", err.Error())
		return
	}

	newState := modelFromAPIUser(updated)
	newState.Password = plan.Password
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid User ID in State", err.Error())
		return
	}

	if err := rbac.DeleteUser(ctx, r.client, userID); err != nil && !rbac.IsNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting User", err.Error())
	}
}

func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// RBAC entities are account-wide, not scoped to an application_id, so
	// import is just the entity's own numeric ID (unlike every
	// alertandrespond resource's "<application_id>/<id>" composite form).
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
