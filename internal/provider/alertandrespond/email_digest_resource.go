package alertandrespond

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

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
)

var (
	_ resource.Resource                = &emailDigestResource{}
	_ resource.ResourceWithConfigure   = &emailDigestResource{}
	_ resource.ResourceWithImportState = &emailDigestResource{}
)

func NewEmailDigestResource() resource.Resource {
	return &emailDigestResource{}
}

type emailDigestResource struct {
	client *client.Client
}

type emailDigestResourceModel struct {
	ID                   types.String         `tfsdk:"id"`
	ApplicationID        types.Int64          `tfsdk:"application_id"`
	Name                 types.String         `tfsdk:"name"`
	Enabled              types.Bool           `tfsdk:"enabled"`
	Frequency            types.Int64          `tfsdk:"frequency"`
	ActionsJSON          jsontypes.Normalized `tfsdk:"actions_json"`
	EventsJSON           jsontypes.Normalized `tfsdk:"events_json"`
	SelectedEntitiesJSON jsontypes.Normalized `tfsdk:"selected_entities_json"`
}

func (r *emailDigestResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_email_digest"
}

func (r *emailDigestResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An AppDynamics email digest: a periodic rollup email binding trigger events on a set of entities to a set of actions. Note: unlike appdynamics_policy, email digests do not support batching actions -- there is no execute_actions_in_batch attribute here.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"application_id": schema.Int64Attribute{
				Required:      true,
				Description:   "ID of the business application this email digest belongs to.",
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
			"frequency": schema.Int64Attribute{
				Required:    true,
				Description: "How often the digest email is sent, in hours (1-168).",
			},
			"actions_json": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Required:    true,
				Description: "JSON array of action references, e.g. [{\"actionName\":\"page-oncall\",\"actionType\":\"EMAIL\"}]. See the Splunk AppDynamics Email Digest API docs for the shape, including optional specifiedEntityActionDetails.",
			},
			"events_json": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Required:    true,
				Description: "JSON object for the Events block (healthRuleEvents / otherEvents / anomalyEvents / customEvents). See the Splunk AppDynamics Email Digest API docs for the shape.",
			},
			"selected_entities_json": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Required:    true,
				Description: "JSON object for the SelectedEntities block, e.g. {\"selectedEntityType\":\"ANY_ENTITY\"}. See the Splunk AppDynamics Email Digest API docs for the shape when selectedEntityType is SPECIFIC_ENTITIES.",
			},
		},
	}
}

func (r *emailDigestResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func apiEmailDigestFromModel(m emailDigestResourceModel) *client.EmailDigest {
	ed := &client.EmailDigest{
		Name:      m.Name.ValueString(),
		Enabled:   m.Enabled.ValueBool(),
		Frequency: m.Frequency.ValueInt64(),
	}
	if !m.ActionsJSON.IsNull() && !m.ActionsJSON.IsUnknown() {
		ed.Actions = json.RawMessage(m.ActionsJSON.ValueString())
	}
	if !m.EventsJSON.IsNull() && !m.EventsJSON.IsUnknown() {
		ed.Events = json.RawMessage(m.EventsJSON.ValueString())
	}
	if !m.SelectedEntitiesJSON.IsNull() && !m.SelectedEntitiesJSON.IsUnknown() {
		ed.SelectedEntities = json.RawMessage(m.SelectedEntitiesJSON.ValueString())
	}
	return ed
}

func modelFromAPIEmailDigest(applicationID int64, ed *client.EmailDigest) emailDigestResourceModel {
	m := emailDigestResourceModel{
		ID:            types.StringValue(strconv.FormatInt(ed.ID, 10)),
		ApplicationID: types.Int64Value(applicationID),
		Name:          types.StringValue(ed.Name),
		Enabled:       types.BoolValue(ed.Enabled),
		Frequency:     types.Int64Value(ed.Frequency),
	}
	if len(ed.Actions) > 0 {
		m.ActionsJSON = jsontypes.NewNormalizedValue(string(ed.Actions))
	} else {
		m.ActionsJSON = jsontypes.NewNormalizedNull()
	}
	if len(ed.Events) > 0 {
		m.EventsJSON = jsontypes.NewNormalizedValue(string(ed.Events))
	} else {
		m.EventsJSON = jsontypes.NewNormalizedNull()
	}
	if len(ed.SelectedEntities) > 0 {
		m.SelectedEntitiesJSON = jsontypes.NewNormalizedValue(string(ed.SelectedEntities))
	} else {
		m.SelectedEntitiesJSON = jsontypes.NewNormalizedNull()
	}
	return m
}

func (r *emailDigestResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan emailDigestResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateEmailDigest(ctx, plan.ApplicationID.ValueInt64(), apiEmailDigestFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Email Digest", err.Error())
		return
	}

	state := modelFromAPIEmailDigest(plan.ApplicationID.ValueInt64(), created)
	// actions_json/events_json/selected_entities_json are Required (not Computed):
	// the API echoes these back with extra server-defaulted fields the plan never
	// specified, so state must keep the plan's own value or Terraform flags an
	// inconsistent result.
	state.ActionsJSON = plan.ActionsJSON
	state.EventsJSON = plan.EventsJSON
	state.SelectedEntitiesJSON = plan.SelectedEntitiesJSON
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *emailDigestResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state emailDigestResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	emailDigestID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Email Digest ID in State", err.Error())
		return
	}

	found, err := r.client.GetEmailDigest(ctx, state.ApplicationID.ValueInt64(), emailDigestID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Email Digest", err.Error())
		return
	}

	newState := modelFromAPIEmailDigest(state.ApplicationID.ValueInt64(), found)
	// See the comment in Create: the API response can never exactly match what a
	// user wrote in config, so keep the prior state's value here instead of the
	// freshly-fetched one to avoid a perpetual, spurious diff on every plan.
	newState.ActionsJSON = state.ActionsJSON
	newState.EventsJSON = state.EventsJSON
	newState.SelectedEntitiesJSON = state.SelectedEntitiesJSON
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *emailDigestResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan emailDigestResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state emailDigestResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	emailDigestID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Email Digest ID in State", err.Error())
		return
	}

	updated, err := r.client.UpdateEmailDigest(ctx, plan.ApplicationID.ValueInt64(), emailDigestID, apiEmailDigestFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Email Digest", err.Error())
		return
	}

	newState := modelFromAPIEmailDigest(plan.ApplicationID.ValueInt64(), updated)
	newState.ActionsJSON = plan.ActionsJSON
	newState.EventsJSON = plan.EventsJSON
	newState.SelectedEntitiesJSON = plan.SelectedEntitiesJSON
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *emailDigestResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state emailDigestResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	emailDigestID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Email Digest ID in State", err.Error())
		return
	}

	if err := r.client.DeleteEmailDigest(ctx, state.ApplicationID.ValueInt64(), emailDigestID); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting Email Digest", err.Error())
	}
}

func (r *emailDigestResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importCompositeID(ctx, req, resp)
}
