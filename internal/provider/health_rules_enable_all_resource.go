package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pmateos/terraform-provider-appdynamics/internal/client"
)

var (
	_ resource.Resource                = &healthRulesEnableAllResource{}
	_ resource.ResourceWithConfigure   = &healthRulesEnableAllResource{}
	_ resource.ResourceWithImportState = &healthRulesEnableAllResource{}
)

func NewHealthRulesEnableAllResource() resource.Resource {
	return &healthRulesEnableAllResource{}
}

type healthRulesEnableAllResource struct {
	client *client.Client
}

type healthRulesEnableAllResourceModel struct {
	ID            types.String `tfsdk:"id"`
	ApplicationID types.Int64  `tfsdk:"application_id"`
}

func (r *healthRulesEnableAllResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_health_rules_enable_all"
}

func (r *healthRulesEnableAllResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Enables every health rule for an AppDynamics business application in one call. " +
			"This models a one-shot bulk action, not a persistent object: creating it calls the " +
			"enable-all endpoint once, and Terraform never detects drift if a rule is individually " +
			"disabled afterward (there's nothing to read back). To re-run enable-all, taint and " +
			"re-apply this resource. Destroying it calls the matching disable-all endpoint.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"application_id": schema.Int64Attribute{
				Required:      true,
				Description:   "ID of the business application to enable all health rules for.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *healthRulesEnableAllResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func (r *healthRulesEnableAllResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan healthRulesEnableAllResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID := plan.ApplicationID.ValueInt64()
	if err := r.client.EnableAllHealthRules(ctx, applicationID); err != nil {
		resp.Diagnostics.AddError("Error Enabling All Health Rules", err.Error())
		return
	}

	plan.ID = types.StringValue(strconv.FormatInt(applicationID, 10))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read is a no-op: "all health rules enabled" isn't a persistent object the
// API exposes for lookup, and re-deriving it from the list endpoint would
// produce false drift the moment anyone disables a single rule out of band.
func (r *healthRulesEnableAllResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state healthRulesEnableAllResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update never actually runs in practice: application_id is the only
// non-computed attribute and it RequiresReplace, so any change goes through
// Delete+Create instead. Implemented to satisfy the resource.Resource interface.
func (r *healthRulesEnableAllResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan healthRulesEnableAllResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *healthRulesEnableAllResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state healthRulesEnableAllResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DisableAllHealthRules(ctx, state.ApplicationID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Error Disabling All Health Rules", err.Error())
	}
}

func (r *healthRulesEnableAllResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	applicationID, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import ID to be a numeric application_id, got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_id"), applicationID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
