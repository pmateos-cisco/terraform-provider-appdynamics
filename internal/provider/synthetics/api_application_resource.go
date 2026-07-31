package synthetics

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
	"github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/synthetics"
)

var (
	_ resource.Resource                = &apiApplicationResource{}
	_ resource.ResourceWithConfigure   = &apiApplicationResource{}
	_ resource.ResourceWithImportState = &apiApplicationResource{}
)

func NewAPIApplicationResource() resource.Resource {
	return &apiApplicationResource{}
}

type apiApplicationResource struct {
	client *client.Client
}

type apiApplicationResourceModel struct {
	ID     types.String `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	AppKey types.String `tfsdk:"app_key"`
}

func (r *apiApplicationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_synthetic_api_application"
}

func (r *apiApplicationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Synthetic API Monitoring application: a lightweight, account-wide container that appdynamics_synthetic_api_job resources are grouped under. Unlike Synthetic Web Monitoring (which requires a pre-existing Browser RUM app set up via the Controller UI), this container has a full create/delete lifecycle via the API (verified live). There is no update endpoint (verified live), so name forces replacement on change.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"app_key": schema.StringAttribute{
				Computed:    true,
				Description: "Server-assigned EUM app key for this application.",
			},
		},
	}
}

func (r *apiApplicationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func modelFromAPIApplication(app *synthetics.APIApplication) apiApplicationResourceModel {
	return apiApplicationResourceModel{
		ID:     types.StringValue(strconv.FormatInt(app.ID, 10)),
		Name:   types.StringValue(app.Name),
		AppKey: stringOrNull(app.AppKey),
	}
}

func (r *apiApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan apiApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID, err := synthetics.CreateAPIApplication(ctx, r.client, plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Synthetic API Application", err.Error())
		return
	}

	created, err := synthetics.GetAPIApplication(ctx, r.client, applicationID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Synthetic API Application After Create", err.Error())
		return
	}

	state := modelFromAPIApplication(created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *apiApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state apiApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Synthetic API Application ID in State", err.Error())
		return
	}

	found, err := synthetics.GetAPIApplication(ctx, r.client, applicationID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Synthetic API Application", err.Error())
		return
	}

	newState := modelFromAPIApplication(found)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

// Update is never actually invoked: name is the only non-Computed attribute
// and it forces replacement, since there is no update endpoint (verified
// live: POST .../updateApplication 404s).
func (r *apiApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan apiApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *apiApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state apiApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Synthetic API Application ID in State", err.Error())
		return
	}

	if err := synthetics.DeleteAPIApplication(ctx, r.client, applicationID); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting Synthetic API Application", err.Error())
	}
}

func (r *apiApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
