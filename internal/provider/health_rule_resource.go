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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client"
)

var (
	_ resource.Resource                = &healthRuleResource{}
	_ resource.ResourceWithConfigure   = &healthRuleResource{}
	_ resource.ResourceWithImportState = &healthRuleResource{}
)

func NewHealthRuleResource() resource.Resource {
	return &healthRuleResource{}
}

type healthRuleResource struct {
	client *client.Client
}

type healthRuleResourceModel struct {
	ID                      types.String         `tfsdk:"id"`
	ApplicationID           types.Int64          `tfsdk:"application_id"`
	Name                    types.String         `tfsdk:"name"`
	Enabled                 types.Bool           `tfsdk:"enabled"`
	UseDataFromLastNMinutes types.Int64          `tfsdk:"use_data_from_last_n_minutes"`
	WaitTimeAfterViolation  types.Int64          `tfsdk:"wait_time_after_violation"`
	ScheduleName            types.String         `tfsdk:"schedule_name"`
	AffectsJSON             jsontypes.Normalized `tfsdk:"affects_json"`
	EvalCriteriasJSON       jsontypes.Normalized `tfsdk:"eval_criterias_json"`
}

func (r *healthRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_health_rule"
}

func (r *healthRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An AppDynamics health rule, defining warning/critical evaluation criteria for an entity in a business application.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"application_id": schema.Int64Attribute{
				Required:      true,
				Description:   "ID of the business application this health rule belongs to.",
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
			"use_data_from_last_n_minutes": schema.Int64Attribute{
				Optional:    true,
				Description: "Evaluation window in minutes (1-360, or multiples of 10).",
			},
			"wait_time_after_violation": schema.Int64Attribute{
				Optional:    true,
				Description: "Minutes to wait after a violation before re-evaluating.",
			},
			"schedule_name": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("Always"),
			},
			"affects_json": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Required:    true,
				Description: "JSON object for the Affects block, e.g. {\"affectedEntityType\":\"TIER_NODE_HARDWARE\",\"affectedEntities\":{\"tierOrNode\":\"TIER_AFFECTED_ENTITIES\",\"affectedTiers\":{\"affectedTierScope\":\"SPECIFIC_TIERS\",\"tiers\":[\"Tier1\"],\"shouldNot\":false}}}. See the Splunk AppDynamics Health Rule API docs for the shape per affectedEntityType.",
			},
			"eval_criterias_json": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Required:    true,
				Description: "JSON object for the EvalCriterias block (criticalCriteria / warningCriteria). See the Splunk AppDynamics Health Rule API docs for the shape.",
			},
		},
	}
}

func (r *healthRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func apiHealthRuleFromModel(m healthRuleResourceModel) *client.HealthRule {
	hr := &client.HealthRule{
		Name:                    m.Name.ValueString(),
		Enabled:                 m.Enabled.ValueBool(),
		UseDataFromLastNMinutes: m.UseDataFromLastNMinutes.ValueInt64(),
		WaitTimeAfterViolation:  m.WaitTimeAfterViolation.ValueInt64(),
		ScheduleName:            m.ScheduleName.ValueString(),
	}
	if !m.AffectsJSON.IsNull() && !m.AffectsJSON.IsUnknown() {
		hr.Affects = json.RawMessage(m.AffectsJSON.ValueString())
	}
	if !m.EvalCriteriasJSON.IsNull() && !m.EvalCriteriasJSON.IsUnknown() {
		hr.EvalCriterias = json.RawMessage(m.EvalCriteriasJSON.ValueString())
	}
	return hr
}

func modelFromAPIHealthRule(applicationID int64, hr *client.HealthRule) healthRuleResourceModel {
	m := healthRuleResourceModel{
		ID:            types.StringValue(strconv.FormatInt(hr.ID, 10)),
		ApplicationID: types.Int64Value(applicationID),
		Name:          types.StringValue(hr.Name),
		Enabled:       types.BoolValue(hr.Enabled),
		ScheduleName:  stringOrNull(hr.ScheduleName),
	}
	if hr.UseDataFromLastNMinutes != 0 {
		m.UseDataFromLastNMinutes = types.Int64Value(hr.UseDataFromLastNMinutes)
	} else {
		m.UseDataFromLastNMinutes = types.Int64Null()
	}
	if hr.WaitTimeAfterViolation != 0 {
		m.WaitTimeAfterViolation = types.Int64Value(hr.WaitTimeAfterViolation)
	} else {
		m.WaitTimeAfterViolation = types.Int64Null()
	}
	if len(hr.Affects) > 0 {
		m.AffectsJSON = jsontypes.NewNormalizedValue(string(hr.Affects))
	} else {
		m.AffectsJSON = jsontypes.NewNormalizedNull()
	}
	if len(hr.EvalCriterias) > 0 {
		m.EvalCriteriasJSON = jsontypes.NewNormalizedValue(string(hr.EvalCriterias))
	} else {
		m.EvalCriteriasJSON = jsontypes.NewNormalizedNull()
	}
	return m
}

func (r *healthRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan healthRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateHealthRule(ctx, plan.ApplicationID.ValueInt64(), apiHealthRuleFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Health Rule", err.Error())
		return
	}

	state := modelFromAPIHealthRule(plan.ApplicationID.ValueInt64(), created)
	// affects_json/eval_criterias_json are Required (not Computed): the API echoes
	// these back with extra server-defaulted fields the plan never specified, so
	// state must keep the plan's own value or Terraform flags an inconsistent result.
	state.AffectsJSON = plan.AffectsJSON
	state.EvalCriteriasJSON = plan.EvalCriteriasJSON
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *healthRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state healthRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	healthRuleID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Health Rule ID in State", err.Error())
		return
	}

	found, err := r.client.GetHealthRule(ctx, state.ApplicationID.ValueInt64(), healthRuleID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Health Rule", err.Error())
		return
	}

	newState := modelFromAPIHealthRule(state.ApplicationID.ValueInt64(), found)
	// The API echoes affects/evalCriterias back with extra server-defaulted fields
	// that will never match what a user wrote in config, so a literal diff against
	// the API response would show a perpetual, spurious update on every plan. Keep
	// the prior state's value instead; drift here would already have surfaced as a
	// remote error when it mattered (e.g. on the next apply).
	newState.AffectsJSON = state.AffectsJSON
	newState.EvalCriteriasJSON = state.EvalCriteriasJSON
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *healthRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan healthRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state healthRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	healthRuleID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Health Rule ID in State", err.Error())
		return
	}

	updated, err := r.client.UpdateHealthRule(ctx, plan.ApplicationID.ValueInt64(), healthRuleID, apiHealthRuleFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Health Rule", err.Error())
		return
	}

	newState := modelFromAPIHealthRule(plan.ApplicationID.ValueInt64(), updated)
	newState.AffectsJSON = plan.AffectsJSON
	newState.EvalCriteriasJSON = plan.EvalCriteriasJSON
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *healthRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state healthRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	healthRuleID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Health Rule ID in State", err.Error())
		return
	}

	if err := r.client.DeleteHealthRule(ctx, state.ApplicationID.ValueInt64(), healthRuleID); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting Health Rule", err.Error())
	}
}

func (r *healthRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importCompositeID(ctx, req, resp)
}
