package alertandrespond

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
)

var (
	_ resource.Resource                = &actionResource{}
	_ resource.ResourceWithConfigure   = &actionResource{}
	_ resource.ResourceWithImportState = &actionResource{}
)

func NewActionResource() resource.Resource {
	return &actionResource{}
}

type actionResource struct {
	client *client.Client
}

type actionResourceModel struct {
	ID            types.String         `tfsdk:"id"`
	ApplicationID types.Int64          `tfsdk:"application_id"`
	Name          types.String         `tfsdk:"name"`
	ActionType    types.String         `tfsdk:"action_type"`
	Notes         types.String         `tfsdk:"notes"`
	ExtraFields   jsontypes.Normalized `tfsdk:"extra_fields"`
}

func (r *actionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_action"
}

func (r *actionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An AppDynamics alert action (email, thread dump, HTTP request, JIRA ticket, etc.), invoked by a policy.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"application_id": schema.Int64Attribute{
				Required:      true,
				Description:   "ID of the business application this action belongs to.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"action_type": schema.StringAttribute{
				Required:    true,
				Description: "One of SMS, EMAIL, CUSTOM_EMAIL, THREAD_DUMP, HTTP_REQUEST, RUN_SCRIPT_ON_NODES, DIAGNOSE_BUSINESS_TRANSACTIONS, CREATE_UPDATE_JIRA.",
			},
			"notes": schema.StringAttribute{
				Optional: true,
			},
			"extra_fields": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Optional:    true,
				Description: "JSON object with the fields specific to action_type (e.g. {\"emails\":[\"a@example.com\"]} for EMAIL, {\"numberOfThreadDumps\":5,\"intervalInMs\":1000} for THREAD_DUMP). See the Splunk AppDynamics Actions API docs for the shape per action_type.",
			},
		},
	}
}

func (r *actionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func apiActionFromModel(m actionResourceModel) *client.Action {
	a := &client.Action{
		Name:       m.Name.ValueString(),
		ActionType: m.ActionType.ValueString(),
		Notes:      m.Notes.ValueString(),
	}
	if !m.ExtraFields.IsNull() && !m.ExtraFields.IsUnknown() {
		a.ExtraFields = json.RawMessage(m.ExtraFields.ValueString())
	}
	return a
}

func modelFromAPIAction(applicationID int64, a *client.Action) actionResourceModel {
	m := actionResourceModel{
		ID:            types.StringValue(strconv.FormatInt(a.ID, 10)),
		ApplicationID: types.Int64Value(applicationID),
		Name:          types.StringValue(a.Name),
		ActionType:    types.StringValue(a.ActionType),
		Notes:         stringOrNull(a.Notes),
	}
	if len(a.ExtraFields) > 0 && string(a.ExtraFields) != "{}" {
		m.ExtraFields = jsontypes.NewNormalizedValue(string(a.ExtraFields))
	} else {
		m.ExtraFields = jsontypes.NewNormalizedNull()
	}
	return m
}

func (r *actionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan actionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateAction(ctx, plan.ApplicationID.ValueInt64(), apiActionFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Action", err.Error())
		return
	}

	state := modelFromAPIAction(plan.ApplicationID.ValueInt64(), created)
	// extra_fields is not Computed: if the API ever echoes it back with extra
	// server-defaulted fields the plan never specified, state must keep the
	// plan's own value or Terraform flags an inconsistent result.
	state.ExtraFields = plan.ExtraFields
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *actionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state actionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	actionID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Action ID in State", err.Error())
		return
	}

	found, err := r.client.GetAction(ctx, state.ApplicationID.ValueInt64(), actionID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Action", err.Error())
		return
	}

	newState := modelFromAPIAction(state.ApplicationID.ValueInt64(), found)
	// See the comment in Create: keep the prior state's value instead of the
	// freshly-fetched one to avoid a perpetual, spurious diff on every plan.
	newState.ExtraFields = state.ExtraFields
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *actionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan actionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state actionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	actionID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Action ID in State", err.Error())
		return
	}

	updated, err := r.client.UpdateAction(ctx, plan.ApplicationID.ValueInt64(), actionID, apiActionFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Action", err.Error())
		return
	}

	newState := modelFromAPIAction(plan.ApplicationID.ValueInt64(), updated)
	newState.ExtraFields = plan.ExtraFields
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *actionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state actionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	actionID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Action ID in State", err.Error())
		return
	}

	if err := r.client.DeleteAction(ctx, state.ApplicationID.ValueInt64(), actionID); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting Action", err.Error())
	}
}

func (r *actionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importCompositeID(ctx, req, resp)
}
