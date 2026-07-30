package alertandrespond

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
)

// stringListToGo converts a (possibly null/unknown) Terraform list of strings
// into a Go []string, treating null and unknown as an empty slice.
func stringListToGo(ctx context.Context, l types.List) ([]string, diag.Diagnostics) {
	if l.IsNull() || l.IsUnknown() {
		return nil, nil
	}
	var out []string
	diags := l.ElementsAs(ctx, &out, false)
	return out, diags
}

// goStringList converts a Go []string into a Terraform list of strings,
// returning a null list for an empty/nil slice.
func goStringList(ctx context.Context, s []string) (types.List, diag.Diagnostics) {
	if len(s) == 0 {
		return types.ListNull(types.StringType), nil
	}
	return types.ListValueFrom(ctx, types.StringType, s)
}

// configureClient extracts the shared *client.Client from ProviderData,
// recording a diagnostic if it's missing or the wrong type. It's called from
// every resource's Configure method.
func configureClient(req resource.ConfigureRequest, resp *resource.ConfigureResponse) *client.Client {
	if req.ProviderData == nil {
		return nil
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. This is a bug in the provider.", req.ProviderData),
		)
		return nil
	}
	return c
}

// importCompositeID parses the "application_id/resource_id" import ID format
// shared by every resource in this provider and writes both attributes into
// state so a subsequent Read can populate the rest.
func importCompositeID(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import ID in the form <application_id>/<resource_id>, got: %q", req.ID),
		)
		return
	}

	applicationID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid application_id in Import Identifier", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_id"), applicationID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
