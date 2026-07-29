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

	"github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client"
)

var (
	_ resource.Resource                = &actionSuppressionResource{}
	_ resource.ResourceWithConfigure   = &actionSuppressionResource{}
	_ resource.ResourceWithImportState = &actionSuppressionResource{}
)

func NewActionSuppressionResource() resource.Resource {
	return &actionSuppressionResource{}
}

type actionSuppressionResource struct {
	client *client.Client
}

type actionSuppressionResourceModel struct {
	ID                      types.String         `tfsdk:"id"`
	ApplicationID           types.Int64          `tfsdk:"application_id"`
	Name                    types.String         `tfsdk:"name"`
	DisableAgentReporting   types.Bool           `tfsdk:"disable_agent_reporting"`
	SuppressionScheduleType types.String         `tfsdk:"suppression_schedule_type"`
	Timezone                types.String         `tfsdk:"timezone"`
	StartTime               types.String         `tfsdk:"start_time"`
	EndTime                 types.String         `tfsdk:"end_time"`
	AffectsJSON             jsontypes.Normalized `tfsdk:"affects_json"`
	RecurringScheduleJSON   jsontypes.Normalized `tfsdk:"recurring_schedule_json"`
	HealthRuleScopeJSON     jsontypes.Normalized `tfsdk:"health_rule_scope_json"`
}

func (r *actionSuppressionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_action_suppression"
}

func (r *actionSuppressionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An AppDynamics action suppression: a schedule during which actions are muted for a scope of entities.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"application_id": schema.Int64Attribute{
				Required:      true,
				Description:   "ID of the business application this action suppression belongs to.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"disable_agent_reporting": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "If true, agents stop reporting metrics during the suppression window.",
			},
			"suppression_schedule_type": schema.StringAttribute{
				Required:    true,
				Description: "ONE_TIME or RECURRING.",
			},
			"timezone": schema.StringAttribute{
				Optional:    true,
				Description: "Timezone ID, e.g. America/Los_Angeles. Defaults to the Controller's timezone if unset.",
			},
			"start_time": schema.StringAttribute{
				Optional:    true,
				Description: "Start of a ONE_TIME suppression window, format yyyy-MM-ddTHH:mm:ss. Not used for RECURRING (see recurring_schedule_json).",
			},
			"end_time": schema.StringAttribute{
				Optional:    true,
				Description: "End of a ONE_TIME suppression window, format yyyy-MM-ddTHH:mm:ss. Not used for RECURRING (see recurring_schedule_json).",
			},
			"affects_json": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Required:    true,
				Description: "JSON object for the Affects block, e.g. {\"affectedInfoType\":\"APPLICATION\"}. See the Splunk AppDynamics Action Suppression API docs for the shape per affectedInfoType.",
			},
			"recurring_schedule_json": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Optional:    true,
				Description: "JSON object for the RecurringSchedule block, required when suppression_schedule_type is RECURRING, e.g. {\"scheduleFrequency\":\"WEEKLY\",\"days\":[\"SATURDAY\",\"SUNDAY\"],\"startTime\":\"09:00\",\"endTime\":\"10:00\"} — same scheduleFrequency shapes as appdynamics_schedule's schedule_configuration.",
			},
			"health_rule_scope_json": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Optional:    true,
				Description: "JSON object for the HealthRuleScope block, e.g. {\"healthRuleScopeType\":\"SPECIFIC_HEALTH_RULES\",\"healthRules\":[\"High CPU Usage\"]}. Omit to apply to all health rules.",
			},
		},
	}
}

func (r *actionSuppressionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func apiActionSuppressionFromModel(m actionSuppressionResourceModel) *client.ActionSuppression {
	as := &client.ActionSuppression{
		Name:                    m.Name.ValueString(),
		DisableAgentReporting:   m.DisableAgentReporting.ValueBool(),
		SuppressionScheduleType: m.SuppressionScheduleType.ValueString(),
		Timezone:                m.Timezone.ValueString(),
		StartTime:               m.StartTime.ValueString(),
		EndTime:                 m.EndTime.ValueString(),
	}
	if !m.AffectsJSON.IsNull() && !m.AffectsJSON.IsUnknown() {
		as.Affects = json.RawMessage(m.AffectsJSON.ValueString())
	}
	if !m.RecurringScheduleJSON.IsNull() && !m.RecurringScheduleJSON.IsUnknown() {
		as.RecurringSchedule = json.RawMessage(m.RecurringScheduleJSON.ValueString())
	}
	if !m.HealthRuleScopeJSON.IsNull() && !m.HealthRuleScopeJSON.IsUnknown() {
		as.HealthRuleScope = json.RawMessage(m.HealthRuleScopeJSON.ValueString())
	}
	return as
}

func modelFromAPIActionSuppression(applicationID int64, as *client.ActionSuppression) actionSuppressionResourceModel {
	m := actionSuppressionResourceModel{
		ID:                      types.StringValue(strconv.FormatInt(as.ID, 10)),
		ApplicationID:           types.Int64Value(applicationID),
		Name:                    types.StringValue(as.Name),
		DisableAgentReporting:   types.BoolValue(as.DisableAgentReporting),
		SuppressionScheduleType: types.StringValue(as.SuppressionScheduleType),
		Timezone:                stringOrNull(as.Timezone),
		StartTime:               stringOrNull(as.StartTime),
		EndTime:                 stringOrNull(as.EndTime),
	}
	if len(as.Affects) > 0 {
		m.AffectsJSON = jsontypes.NewNormalizedValue(string(as.Affects))
	} else {
		m.AffectsJSON = jsontypes.NewNormalizedNull()
	}
	if len(as.RecurringSchedule) > 0 && string(as.RecurringSchedule) != "null" {
		m.RecurringScheduleJSON = jsontypes.NewNormalizedValue(string(as.RecurringSchedule))
	} else {
		m.RecurringScheduleJSON = jsontypes.NewNormalizedNull()
	}
	if len(as.HealthRuleScope) > 0 && string(as.HealthRuleScope) != "null" {
		m.HealthRuleScopeJSON = jsontypes.NewNormalizedValue(string(as.HealthRuleScope))
	} else {
		m.HealthRuleScopeJSON = jsontypes.NewNormalizedNull()
	}
	return m
}

// keepPassthroughJSON preserves the plan's/prior state's own value for the
// JSON passthrough attributes instead of the API's echoed-back response,
// which includes extra server-defaulted fields (e.g. suppressionMaintenanceType
// at the top level, and possibly nested defaults) that were never in the
// original request. Required (non-Computed) attributes must match the plan
// exactly after Create/Update or Terraform flags an inconsistent result;
// Optional non-Computed attributes have the same requirement. Read compares
// against config too, so reusing the API response there would show a
// perpetual, spurious diff on every plan.
func keepPassthroughJSON(dst *actionSuppressionResourceModel, src actionSuppressionResourceModel) {
	dst.AffectsJSON = src.AffectsJSON
	dst.RecurringScheduleJSON = src.RecurringScheduleJSON
	dst.HealthRuleScopeJSON = src.HealthRuleScopeJSON
}

func (r *actionSuppressionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan actionSuppressionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateActionSuppression(ctx, plan.ApplicationID.ValueInt64(), apiActionSuppressionFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Action Suppression", err.Error())
		return
	}

	state := modelFromAPIActionSuppression(plan.ApplicationID.ValueInt64(), created)
	keepPassthroughJSON(&state, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *actionSuppressionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state actionSuppressionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	actionSuppressionID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Action Suppression ID in State", err.Error())
		return
	}

	found, err := r.client.GetActionSuppression(ctx, state.ApplicationID.ValueInt64(), actionSuppressionID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Action Suppression", err.Error())
		return
	}

	newState := modelFromAPIActionSuppression(state.ApplicationID.ValueInt64(), found)
	keepPassthroughJSON(&newState, state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *actionSuppressionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan actionSuppressionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state actionSuppressionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	actionSuppressionID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Action Suppression ID in State", err.Error())
		return
	}

	updated, err := r.client.UpdateActionSuppression(ctx, plan.ApplicationID.ValueInt64(), actionSuppressionID, apiActionSuppressionFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Action Suppression", err.Error())
		return
	}

	newState := modelFromAPIActionSuppression(plan.ApplicationID.ValueInt64(), updated)
	keepPassthroughJSON(&newState, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *actionSuppressionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state actionSuppressionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	actionSuppressionID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Action Suppression ID in State", err.Error())
		return
	}

	if err := r.client.DeleteActionSuppression(ctx, state.ApplicationID.ValueInt64(), actionSuppressionID); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting Action Suppression", err.Error())
	}
}

func (r *actionSuppressionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importCompositeID(ctx, req, resp)
}
