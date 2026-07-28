package provider

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pmateos/terraform-provider-appdynamics/internal/client"
)

var (
	_ resource.Resource                = &policyResource{}
	_ resource.ResourceWithConfigure   = &policyResource{}
	_ resource.ResourceWithImportState = &policyResource{}
)

func NewPolicyResource() resource.Resource {
	return &policyResource{}
}

type policyResource struct {
	client *client.Client
}

type policyResourceModel struct {
	ID                    types.String         `tfsdk:"id"`
	ApplicationID         types.Int64          `tfsdk:"application_id"`
	Name                  types.String         `tfsdk:"name"`
	Enabled               types.Bool           `tfsdk:"enabled"`
	ExecuteActionsInBatch types.Bool           `tfsdk:"execute_actions_in_batch"`
	ActionsJSON           jsontypes.Normalized `tfsdk:"actions_json"`
	EventsJSON            jsontypes.Normalized `tfsdk:"events_json"`
	SelectedEntitiesJSON  jsontypes.Normalized `tfsdk:"selected_entities_json"`
}

func (r *policyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

func (r *policyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An AppDynamics alert policy: binds trigger events on a set of entities to the actions to run.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"application_id": schema.Int64Attribute{
				Required:      true,
				Description:   "ID of the business application this policy belongs to.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"execute_actions_in_batch": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"actions_json": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Required:    true,
				Description: "JSON array of action references, e.g. [{\"actionName\":\"page-oncall\",\"actionType\":\"EMAIL\"}]. See the Splunk AppDynamics Policy API docs for the shape, including optional specifiedEntityActionDetails.",
			},
			"events_json": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Required:    true,
				Description: "JSON object for the Events block (healthRuleEvents / otherEvents / anomalyEvents / customEvents). See the Splunk AppDynamics Policy API docs for the shape.",
			},
			"selected_entities_json": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Required:    true,
				Description: "JSON object for the SelectedEntities block, e.g. {\"selectedEntityType\":\"ANY_ENTITY\"}. See the Splunk AppDynamics Policy API docs for the shape when selectedEntityType is SPECIFIC_ENTITIES.",
			},
		},
	}
}

func (r *policyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func apiPolicyFromModel(m policyResourceModel) *client.Policy {
	p := &client.Policy{
		Name:                  m.Name.ValueString(),
		Enabled:               m.Enabled.ValueBool(),
		ExecuteActionsInBatch: m.ExecuteActionsInBatch.ValueBool(),
	}
	if !m.ActionsJSON.IsNull() && !m.ActionsJSON.IsUnknown() {
		p.Actions = json.RawMessage(m.ActionsJSON.ValueString())
	}
	if !m.EventsJSON.IsNull() && !m.EventsJSON.IsUnknown() {
		p.Events = json.RawMessage(m.EventsJSON.ValueString())
	}
	if !m.SelectedEntitiesJSON.IsNull() && !m.SelectedEntitiesJSON.IsUnknown() {
		p.SelectedEntities = json.RawMessage(m.SelectedEntitiesJSON.ValueString())
	}
	return p
}

func modelFromAPIPolicy(applicationID int64, p *client.Policy) policyResourceModel {
	m := policyResourceModel{
		ID:                    types.StringValue(strconv.FormatInt(p.ID, 10)),
		ApplicationID:         types.Int64Value(applicationID),
		Name:                  types.StringValue(p.Name),
		Enabled:               types.BoolValue(p.Enabled),
		ExecuteActionsInBatch: types.BoolValue(p.ExecuteActionsInBatch),
	}
	if len(p.Actions) > 0 {
		m.ActionsJSON = jsontypes.NewNormalizedValue(string(p.Actions))
	} else {
		m.ActionsJSON = jsontypes.NewNormalizedNull()
	}
	if len(p.Events) > 0 {
		m.EventsJSON = jsontypes.NewNormalizedValue(string(p.Events))
	} else {
		m.EventsJSON = jsontypes.NewNormalizedNull()
	}
	if len(p.SelectedEntities) > 0 {
		m.SelectedEntitiesJSON = jsontypes.NewNormalizedValue(string(p.SelectedEntities))
	} else {
		m.SelectedEntitiesJSON = jsontypes.NewNormalizedNull()
	}
	return m
}

func (r *policyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan policyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreatePolicy(ctx, plan.ApplicationID.ValueInt64(), apiPolicyFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Policy", err.Error())
		return
	}

	state := modelFromAPIPolicy(plan.ApplicationID.ValueInt64(), created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *policyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state policyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policyID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Policy ID in State", err.Error())
		return
	}

	found, err := r.client.GetPolicy(ctx, state.ApplicationID.ValueInt64(), policyID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Policy", err.Error())
		return
	}

	newState := modelFromAPIPolicy(state.ApplicationID.ValueInt64(), found)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *policyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan policyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state policyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policyID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Policy ID in State", err.Error())
		return
	}

	updated, err := r.client.UpdatePolicy(ctx, plan.ApplicationID.ValueInt64(), policyID, apiPolicyFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Policy", err.Error())
		return
	}

	newState := modelFromAPIPolicy(plan.ApplicationID.ValueInt64(), updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *policyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state policyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policyID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Policy ID in State", err.Error())
		return
	}

	if err := r.client.DeletePolicy(ctx, state.ApplicationID.ValueInt64(), policyID); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting Policy", err.Error())
	}
}

func (r *policyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importCompositeID(ctx, req, resp)
}
