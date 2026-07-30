package alertandrespond

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	client "github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client/alertandrespond"
)

var (
	_ datasource.DataSource              = &emailDigestsDataSource{}
	_ datasource.DataSourceWithConfigure = &emailDigestsDataSource{}
)

func NewEmailDigestsDataSource() datasource.DataSource {
	return &emailDigestsDataSource{}
}

type emailDigestsDataSource struct {
	client *client.Client
}

type emailDigestSummaryModel struct {
	ID      types.String `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	Enabled types.Bool   `tfsdk:"enabled"`
}

type emailDigestsDataSourceModel struct {
	ApplicationID types.Int64               `tfsdk:"application_id"`
	EmailDigests  []emailDigestSummaryModel `tfsdk:"email_digests"`
}

func (d *emailDigestsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_email_digests"
}

func (d *emailDigestsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the email digests defined for an AppDynamics business application.",
		Attributes: map[string]schema.Attribute{
			"application_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the business application to list email digests for.",
			},
			"email_digests": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The email digests defined for the application.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Email digest ID.",
						},
						"name": schema.StringAttribute{
							Computed: true,
						},
						"enabled": schema.BoolAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *emailDigestsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *emailDigestsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config emailDigestsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := d.client.ListEmailDigests(ctx, config.ApplicationID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Email Digests", err.Error())
		return
	}

	digests := make([]emailDigestSummaryModel, 0, len(found))
	for _, ed := range found {
		digests = append(digests, emailDigestSummaryModel{
			ID:      types.StringValue(strconv.FormatInt(ed.ID, 10)),
			Name:    types.StringValue(ed.Name),
			Enabled: types.BoolValue(ed.Enabled),
		})
	}

	state := emailDigestsDataSourceModel{
		ApplicationID: config.ApplicationID,
		EmailDigests:  digests,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
