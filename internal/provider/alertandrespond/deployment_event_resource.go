package alertandrespond

import (
	"context"
	"net/url"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
)

var (
	_ resource.Resource              = &deploymentEventResource{}
	_ resource.ResourceWithConfigure = &deploymentEventResource{}
)

func NewDeploymentEventResource() resource.Resource {
	return &deploymentEventResource{}
}

type deploymentEventResource struct {
	client *client.Client
}

type deploymentEventResourceModel struct {
	ID            types.String `tfsdk:"id"`
	ApplicationID types.Int64  `tfsdk:"application_id"`
	Summary       types.String `tfsdk:"summary"`
	Comment       types.String `tfsdk:"comment"`
	Severity      types.String `tfsdk:"severity"`
}

func (r *deploymentEventResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment_event"
}

func (r *deploymentEventResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates an APPLICATION_DEPLOYMENT event, for tracking application upgrades and " +
			"releases in AppDynamics. Events are immutable log entries: the underlying API has no " +
			"support for reading a single event back by ID, updating one, or deleting one. Because of " +
			"that, this resource is create-only — any configuration change replaces it with a brand new " +
			"event (the old one still exists in the AppDynamics event log), `Read` never contacts the " +
			"API (there's nothing reliable to read back), and `terraform destroy` only removes it from " +
			"Terraform state, with a warning, since the event itself cannot be deleted from AppDynamics.",
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
		},
	}
}

func (r *deploymentEventResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func (r *deploymentEventResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan deploymentEventResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	form := url.Values{}
	form.Set("eventtype", "APPLICATION_DEPLOYMENT")
	form.Set("summary", plan.Summary.ValueString())
	form.Set("severity", plan.Severity.ValueString())
	if !plan.Comment.IsNull() {
		form.Set("comment", plan.Comment.ValueString())
	}

	id, err := r.client.CreateEvent(ctx, plan.ApplicationID.ValueInt64(), form)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Deployment Event", err.Error())
		return
	}

	plan.ID = types.StringValue(strconv.FormatInt(id, 10))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read is a no-op: events are immutable log entries with no "get by ID"
// endpoint, so there's nothing reliable to refresh from the API.
func (r *deploymentEventResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state deploymentEventResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update never actually runs: every attribute RequiresReplace, so any change
// goes through Delete+Create instead. Implemented to satisfy the
// resource.Resource interface.
func (r *deploymentEventResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan deploymentEventResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete cannot actually remove the event from AppDynamics -- the Events API
// has no delete endpoint. It only removes the resource from Terraform state.
func (r *deploymentEventResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning(
		"Event Not Deleted in AppDynamics",
		"The AppDynamics Events API does not support deleting events. This event remains in the "+
			"AppDynamics event log; it has only been removed from Terraform state.",
	)
}
