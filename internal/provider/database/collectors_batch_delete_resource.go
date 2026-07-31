package database

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
	"github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/database"
)

var (
	_ resource.Resource              = &collectorsBatchDeleteResource{}
	_ resource.ResourceWithConfigure = &collectorsBatchDeleteResource{}
)

func NewCollectorsBatchDeleteResource() resource.Resource {
	return &collectorsBatchDeleteResource{}
}

type collectorsBatchDeleteResource struct {
	client *client.Client
}

type collectorsBatchDeleteResourceModel struct {
	ID           types.String  `tfsdk:"id"`
	CollectorIDs []types.Int64 `tfsdk:"collector_ids"`
}

func (r *collectorsBatchDeleteResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_collectors_batch_delete"
}

func (r *collectorsBatchDeleteResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Permanently deletes a specific set of Database Visibility collectors in one call, via the " +
			"(undocumented alongside the regular per-ID delete, but present) batchDelete endpoint. This models a " +
			"one-shot, irreversible bulk action, not a persistent object: creating it deletes the listed " +
			"collector_ids immediately, and terraform destroy is a no-op (there is no way to un-delete a " +
			"collector) -- it only removes this resource from Terraform state, with a warning. Prefer managing " +
			"individual collectors with appdynamics_database_collector unless you specifically need to delete " +
			"several as one atomic call.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"collector_ids": schema.ListAttribute{
				Required:      true,
				ElementType:   types.Int64Type,
				PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()},
				Description:   "IDs of the collectors to delete.",
			},
		},
	}
}

func (r *collectorsBatchDeleteResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func (r *collectorsBatchDeleteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan collectorsBatchDeleteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ids := make([]int64, 0, len(plan.CollectorIDs))
	for _, v := range plan.CollectorIDs {
		ids = append(ids, v.ValueInt64())
	}

	if err := database.BatchDeleteCollectors(ctx, r.client, ids); err != nil {
		resp.Diagnostics.AddError("Error Batch-Deleting Database Collectors", err.Error())
		return
	}

	plan.ID = types.StringValue(batchDeleteID(ids))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read is a no-op: the collectors are already gone by the time this resource
// exists, so there's nothing left to read back or detect drift against.
func (r *collectorsBatchDeleteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state collectorsBatchDeleteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update never actually runs: collector_ids is the only non-computed
// attribute and it forces replacement, so any change goes through
// Delete+Create instead. Implemented to satisfy the resource.Resource
// interface.
func (r *collectorsBatchDeleteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan collectorsBatchDeleteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete cannot actually undo a batch delete -- there is no un-delete API --
// so this only drops the resource from Terraform state, with a warning,
// same as appdynamics_custom_event/appdynamics_deployment_event's Delete.
func (r *collectorsBatchDeleteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning(
		"Database Collectors Not Restored",
		"appdynamics_database_collectors_batch_delete only removes this resource from Terraform state on destroy. "+
			"The collectors it deleted on create cannot be restored via the API.",
	)
}

func batchDeleteID(ids []int64) string {
	sorted := append([]int64(nil), ids...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	parts := make([]string, len(sorted))
	for i, id := range sorted {
		parts[i] = fmt.Sprintf("%d", id)
	}
	return strings.Join(parts, ",")
}
