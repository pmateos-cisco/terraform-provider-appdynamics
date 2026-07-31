package database

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
	"github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/database"
)

var (
	_ resource.Resource                = &collectorResource{}
	_ resource.ResourceWithConfigure   = &collectorResource{}
	_ resource.ResourceWithImportState = &collectorResource{}
)

func NewCollectorResource() resource.Resource {
	return &collectorResource{}
}

type collectorResource struct {
	client *client.Client
}

type collectorResourceModel struct {
	ID              types.String         `tfsdk:"id"`
	Type            types.String         `tfsdk:"type"`
	Name            types.String         `tfsdk:"name"`
	Hostname        types.String         `tfsdk:"hostname"`
	Port            types.Int64          `tfsdk:"port"`
	Username        types.String         `tfsdk:"username"`
	Password        types.String         `tfsdk:"password"`
	AgentName       types.String         `tfsdk:"agent_name"`
	Enabled         types.Bool           `tfsdk:"enabled"`
	ExtraConfigJSON jsontypes.Normalized `tfsdk:"extra_config_json"`
	Version         types.Int64          `tfsdk:"version"`
}

func (r *collectorResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_collector"
}

func (r *collectorResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An AppDynamics Database Visibility collector: the configuration a Database Agent uses to monitor a specific database server. Account-wide (not scoped to a business application).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"type": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Database type, e.g. MYSQL, ORACLE, POSTGRESQL, MSSQL. Immutable (verified live: the API rejects any update that changes it with \"Database Type cannot be modified!\").",
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"hostname": schema.StringAttribute{
				Required: true,
			},
			"port": schema.Int64Attribute{
				Required: true,
			},
			"username": schema.StringAttribute{
				Required: true,
			},
			"password": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Write-only: the API always redacts this in every response, so it cannot be verified or refreshed from the Controller -- Terraform's own state is the only record of what was last sent.",
			},
			"agent_name": schema.StringAttribute{
				Required:    true,
				Description: "Name of an existing Database Agent to run this collector (verified live: this must reference an already-registered agent -- the API rejects unknown names with \"Agent Name: ... does not exist in the database\" and there is no API to create one).",
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the collector actively monitors. Defaults to false (verified live: the API leaves a newly created collector disabled unless this is explicitly set true).",
			},
			"extra_config_json": schema.StringAttribute{
				CustomType: jsontypes.NormalizedType{},
				Optional:   true,
				Description: "JSON object for every database-type-specific setting not modeled as its own attribute here (SSL, CyberArk, JDBC connection properties, custom metrics/events, sub-configs, and dozens of others -- see the Splunk AppDynamics Database Visibility API docs for the full field list). " +
					"The update endpoint requires the complete collector config on every request (verified live: partial updates are rejected), so this provider always resends whatever was last read here alongside the typed attributes above.",
			},
			"version": schema.Int64Attribute{
				Computed:    true,
				Description: "Version number as returned by the API. Verified live: this does not actually increment on update, unlike similar version fields elsewhere in the AppDynamics APIs.",
			},
		},
	}
}

func (r *collectorResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func apiCollectorFromModel(m collectorResourceModel) *database.Collector {
	col := &database.Collector{
		Type:      m.Type.ValueString(),
		Name:      m.Name.ValueString(),
		Hostname:  m.Hostname.ValueString(),
		Port:      int(m.Port.ValueInt64()),
		Username:  m.Username.ValueString(),
		Password:  m.Password.ValueString(),
		AgentName: m.AgentName.ValueString(),
		Enabled:   m.Enabled.ValueBool(),
	}
	if !m.ExtraConfigJSON.IsNull() && !m.ExtraConfigJSON.IsUnknown() {
		col.Extra = []byte(m.ExtraConfigJSON.ValueString())
	}
	return col
}

func modelFromAPICollector(col *database.Collector) collectorResourceModel {
	m := collectorResourceModel{
		ID:        types.StringValue(strconv.FormatInt(col.ID, 10)),
		Type:      types.StringValue(col.Type),
		Name:      types.StringValue(col.Name),
		Hostname:  types.StringValue(col.Hostname),
		Port:      types.Int64Value(int64(col.Port)),
		Username:  types.StringValue(col.Username),
		AgentName: types.StringValue(col.AgentName),
		Enabled:   types.BoolValue(col.Enabled),
		Version:   types.Int64Value(int64(col.Version)),
	}
	if len(col.Extra) > 0 {
		m.ExtraConfigJSON = jsontypes.NewNormalizedValue(string(col.Extra))
	} else {
		m.ExtraConfigJSON = jsontypes.NewNormalizedNull()
	}
	return m
}

func (r *collectorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan collectorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	collectorID, err := database.CreateCollector(ctx, r.client, apiCollectorFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Database Collector", err.Error())
		return
	}

	created, err := database.GetCollector(ctx, r.client, collectorID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Database Collector After Create", err.Error())
		return
	}

	state := modelFromAPICollector(created)
	// password is never echoed back (always redacted) and extra_config_json
	// is not Computed, so keep the plan's own values or Terraform flags an
	// inconsistent result.
	state.Password = plan.Password
	state.ExtraConfigJSON = plan.ExtraConfigJSON
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *collectorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state collectorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	collectorID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Database Collector ID in State", err.Error())
		return
	}

	found, err := database.GetCollector(ctx, r.client, collectorID)
	if err != nil {
		if database.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Database Collector", err.Error())
		return
	}

	newState := modelFromAPICollector(found)
	// See the comment in Create: keep the prior state's own value instead of
	// the freshly-fetched one to avoid a perpetual, spurious diff on every
	// plan.
	newState.Password = state.Password
	newState.ExtraConfigJSON = state.ExtraConfigJSON
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *collectorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan collectorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state collectorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	collectorID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Database Collector ID in State", err.Error())
		return
	}

	apiCollector := apiCollectorFromModel(plan)
	apiCollector.ID = collectorID
	updated, err := database.UpdateCollector(ctx, r.client, apiCollector)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Database Collector", err.Error())
		return
	}

	newState := modelFromAPICollector(updated)
	newState.Password = plan.Password
	newState.ExtraConfigJSON = plan.ExtraConfigJSON
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *collectorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state collectorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	collectorID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Database Collector ID in State", err.Error())
		return
	}

	if err := database.DeleteCollector(ctx, r.client, collectorID); err != nil && !database.IsNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting Database Collector", err.Error())
	}
}

func (r *collectorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
