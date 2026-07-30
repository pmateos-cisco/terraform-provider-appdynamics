package alertandrespond

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
)

var (
	_ datasource.DataSource              = &emailDigestDataSource{}
	_ datasource.DataSourceWithConfigure = &emailDigestDataSource{}
)

func NewEmailDigestDataSource() datasource.DataSource {
	return &emailDigestDataSource{}
}

type emailDigestDataSource struct {
	client *client.Client
}

type emailDigestDataSourceModel struct {
	ApplicationID        types.Int64          `tfsdk:"application_id"`
	EmailDigestID        types.Int64          `tfsdk:"email_digest_id"`
	Name                 types.String         `tfsdk:"name"`
	Enabled              types.Bool           `tfsdk:"enabled"`
	Frequency            types.Int64          `tfsdk:"frequency"`
	ActionsJSON          jsontypes.Normalized `tfsdk:"actions_json"`
	EventsJSON           jsontypes.Normalized `tfsdk:"events_json"`
	SelectedEntitiesJSON jsontypes.Normalized `tfsdk:"selected_entities_json"`
}

func (d *emailDigestDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_email_digest"
}

func (d *emailDigestDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the details of a specific AppDynamics email digest by ID. Use the appdynamics_email_digests data source to discover email_digest_id values.",
		Attributes: map[string]schema.Attribute{
			"application_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the business application this email digest belongs to.",
			},
			"email_digest_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the email digest to retrieve.",
			},
			"name": schema.StringAttribute{
				Computed: true,
			},
			"enabled": schema.BoolAttribute{
				Computed: true,
			},
			"frequency": schema.Int64Attribute{
				Computed:    true,
				Description: "How often the digest email is sent, in hours.",
			},
			"actions_json": schema.StringAttribute{
				CustomType: jsontypes.NormalizedType{},
				Computed:   true,
			},
			"events_json": schema.StringAttribute{
				CustomType: jsontypes.NormalizedType{},
				Computed:   true,
			},
			"selected_entities_json": schema.StringAttribute{
				CustomType: jsontypes.NormalizedType{},
				Computed:   true,
			},
		},
	}
}

func (d *emailDigestDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. This is a bug in the provider.", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *emailDigestDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config emailDigestDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := d.client.GetEmailDigest(ctx, config.ApplicationID.ValueInt64(), config.EmailDigestID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Email Digest", err.Error())
		return
	}

	state := emailDigestDataSourceModel{
		ApplicationID: config.ApplicationID,
		EmailDigestID: config.EmailDigestID,
		Name:          types.StringValue(found.Name),
		Enabled:       types.BoolValue(found.Enabled),
		Frequency:     types.Int64Value(found.Frequency),
	}
	if len(found.Actions) > 0 {
		state.ActionsJSON = jsontypes.NewNormalizedValue(string(found.Actions))
	} else {
		state.ActionsJSON = jsontypes.NewNormalizedNull()
	}
	if len(found.Events) > 0 {
		state.EventsJSON = jsontypes.NewNormalizedValue(string(found.Events))
	} else {
		state.EventsJSON = jsontypes.NewNormalizedNull()
	}
	if len(found.SelectedEntities) > 0 {
		state.SelectedEntitiesJSON = jsontypes.NewNormalizedValue(string(found.SelectedEntities))
	} else {
		state.SelectedEntitiesJSON = jsontypes.NewNormalizedNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
