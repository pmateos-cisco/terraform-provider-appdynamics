// Package provider implements the terraform-plugin-framework Provider for
// Splunk AppDynamics.
package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pmateos-cisco/terraform-provider-appdynamics/internal/client"
)

// New returns a factory for the AppDynamics provider, suitable for passing to
// providerserver.Serve. version is injected at build time via -ldflags and
// surfaced through the provider's Metadata response.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &appdynamicsProvider{version: version}
	}
}

type appdynamicsProvider struct {
	version string
}

type providerModel struct {
	ControllerURL types.String `tfsdk:"controller_url"`
	ClientID      types.String `tfsdk:"client_id"`
	ClientSecret  types.String `tfsdk:"client_secret"`
}

func (p *appdynamicsProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "appdynamics"
	resp.Version = p.version
}

func (p *appdynamicsProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages Splunk AppDynamics alerting configuration (health rules, policies, actions, schedules) via the Controller Platform APIs.",
		Attributes: map[string]schema.Attribute{
			"controller_url": schema.StringAttribute{
				Optional:    true,
				Description: "AppDynamics Controller base URL, e.g. https://mycompany.saas.appdynamics.com. May also be set via the APPD_CONTROLLER_URL environment variable.",
			},
			"client_id": schema.StringAttribute{
				Optional:    true,
				Description: "OAuth API client ID, in the form <api_client_name>@<account_name>. May also be set via the APPD_CLIENT_ID environment variable.",
			},
			"client_secret": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "OAuth API client secret. May also be set via the APPD_CLIENT_SECRET environment variable.",
			},
		},
	}
}

func (p *appdynamicsProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	controllerURL := os.Getenv("APPD_CONTROLLER_URL")
	if !config.ControllerURL.IsNull() {
		controllerURL = config.ControllerURL.ValueString()
	}
	if controllerURL == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("controller_url"),
			"Missing AppDynamics Controller URL",
			"Set the controller_url attribute or the APPD_CONTROLLER_URL environment variable.",
		)
	}

	clientID := os.Getenv("APPD_CLIENT_ID")
	if !config.ClientID.IsNull() {
		clientID = config.ClientID.ValueString()
	}
	if clientID == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("client_id"),
			"Missing AppDynamics OAuth Client ID",
			"Set the client_id attribute or the APPD_CLIENT_ID environment variable.",
		)
	}

	clientSecret := os.Getenv("APPD_CLIENT_SECRET")
	if !config.ClientSecret.IsNull() {
		clientSecret = config.ClientSecret.ValueString()
	}
	if clientSecret == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("client_secret"),
			"Missing AppDynamics OAuth Client Secret",
			"Set the client_secret attribute or the APPD_CLIENT_SECRET environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	c := client.New(controllerURL, clientID, clientSecret)
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *appdynamicsProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewScheduleResource,
		NewActionResource,
		NewHealthRuleResource,
		NewHealthRulesEnableAllResource,
		NewPolicyResource,
		NewActionSuppressionResource,
	}
}

func (p *appdynamicsProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewHealthRulesDataSource,
		NewHealthRuleDataSource,
		NewActionsDataSource,
		NewActionSuppressionsDataSource,
		NewActionSuppressionDataSource,
		NewPoliciesDataSource,
		NewPolicyDataSource,
	}
}
