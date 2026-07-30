package alertandrespond

import (
	"context"
	"net/url"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
)

var (
	_ resource.Resource              = &customEventResource{}
	_ resource.ResourceWithConfigure = &customEventResource{}
)

func NewCustomEventResource() resource.Resource {
	return &customEventResource{}
}

type customEventResource struct {
	client *client.Client
}

type customEventResourceModel struct {
	ID              types.String `tfsdk:"id"`
	ApplicationID   types.Int64  `tfsdk:"application_id"`
	Summary         types.String `tfsdk:"summary"`
	Comment         types.String `tfsdk:"comment"`
	Severity        types.String `tfsdk:"severity"`
	CustomEventType types.String `tfsdk:"custom_event_type"`
	Node            types.String `tfsdk:"node"`
	Tier            types.String `tfsdk:"tier"`
	BT              types.String `tfsdk:"bt"`
	Properties      types.Map    `tfsdk:"properties"`
}

func (r *customEventResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_event"
}

func (r *customEventResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates a custom AppDynamics event, visible in event viewers and dashboards. " +
			"Events are immutable log entries: the underlying API has no support for reading a single " +
			"event back by ID, updating one, or deleting one. Because of that, this resource is " +
			"create-only — any configuration change replaces it with a brand new event (the old one " +
			"still exists in the AppDynamics event log), `Read` never contacts the API (there's nothing " +
			"reliable to read back), and `terraform destroy` only removes it from Terraform state, with " +
			"a warning, since the event itself cannot be deleted from AppDynamics.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The created event's ID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"application_id": schema.Int64Attribute{
				Required:      true,
				Description:   "ID of the business application to create the event on.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"summary": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"comment": schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"severity": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Default:       stringdefault.StaticString("INFO"),
				Description:   "INFO, WARN, or ERROR.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"custom_event_type": schema.StringAttribute{
				Required:      true,
				Description:   "A user-defined event subtype, e.g. \"Deployment\", \"ConfigChange\".",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"node": schema.StringAttribute{
				Optional:      true,
				Description:   "Name of the node this event applies to, if any.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"tier": schema.StringAttribute{
				Optional:      true,
				Description:   "Name of the tier this event applies to, if any.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"bt": schema.StringAttribute{
				Optional:      true,
				Description:   "Name of the business transaction this event applies to, if any.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"properties": schema.MapAttribute{
				ElementType:   types.StringType,
				Optional:      true,
				Description:   "Arbitrary key/value properties attached to the event.",
				PlanModifiers: []planmodifier.Map{mapplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *customEventResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func (r *customEventResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan customEventResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	form := url.Values{}
	form.Set("eventtype", "CUSTOM")
	form.Set("summary", plan.Summary.ValueString())
	form.Set("severity", plan.Severity.ValueString())
	form.Set("customeventtype", plan.CustomEventType.ValueString())
	if !plan.Comment.IsNull() {
		form.Set("comment", plan.Comment.ValueString())
	}
	if !plan.Node.IsNull() {
		form.Set("node", plan.Node.ValueString())
	}
	if !plan.Tier.IsNull() {
		form.Set("tier", plan.Tier.ValueString())
	}
	if !plan.BT.IsNull() {
		form.Set("bt", plan.BT.ValueString())
	}
	if !plan.Properties.IsNull() && !plan.Properties.IsUnknown() {
		var props map[string]string
		resp.Diagnostics.Append(plan.Properties.ElementsAs(ctx, &props, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for k, v := range props {
			form.Add("propertynames", k)
			form.Add("propertyvalues", v)
		}
	}

	id, err := r.client.CreateEvent(ctx, plan.ApplicationID.ValueInt64(), form)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Custom Event", err.Error())
		return
	}

	plan.ID = types.StringValue(strconv.FormatInt(id, 10))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read is a no-op: events are immutable log entries with no "get by ID"
// endpoint, so there's nothing reliable to refresh from the API.
func (r *customEventResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state customEventResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update never actually runs: every attribute RequiresReplace, so any change
// goes through Delete+Create instead. Implemented to satisfy the
// resource.Resource interface.
func (r *customEventResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan customEventResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete cannot actually remove the event from AppDynamics -- the Events API
// has no delete endpoint. It only removes the resource from Terraform state.
func (r *customEventResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning(
		"Event Not Deleted in AppDynamics",
		"The AppDynamics Events API does not support deleting events. This event remains in the "+
			"AppDynamics event log; it has only been removed from Terraform state.",
	)
}
