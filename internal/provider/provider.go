// Copyright (c) RacterMX
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-ractermx/internal/client"
	"terraform-provider-ractermx/internal/datasources"
	"terraform-provider-ractermx/internal/resources"
)

// Ensure RactermxProvider satisfies the provider.Provider interface.
var _ provider.Provider = &RactermxProvider{}

// RactermxProvider defines the provider implementation.
type RactermxProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// RactermxProviderModel describes the provider data model.
type RactermxProviderModel struct {
	ApiKey  types.String `tfsdk:"api_key"`
	BaseUrl types.String `tfsdk:"base_url"`
}

func (p *RactermxProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "ractermx"
	resp.Version = p.version
}

func (p *RactermxProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with the RacterMX API.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Description: "The API key for authenticating with the RacterMX API. " +
					"Can also be set via the RACTERMX_API_KEY environment variable.",
				Optional:  true,
				Sensitive: true,
			},
			"base_url": schema.StringAttribute{
				Description: "The base URL of the RacterMX API. Defaults to https://ractermx.com.",
				Optional:    true,
			},
		},
	}
}

func (p *RactermxProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config RactermxProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If api_key is unknown (e.g., depends on another resource), we cannot
	// configure the client yet. Add a warning and return.
	if config.ApiKey.IsUnknown() {
		resp.Diagnostics.AddWarning(
			"Unknown RacterMX API Key",
			"The provider cannot create the RacterMX API client because the api_key is not yet known. "+
				"This is expected if the api_key depends on another resource. "+
				"Resources and data sources that depend on this provider will not be available until the api_key is resolved.",
		)
		return
	}

	// Resolve api_key: HCL value takes precedence, then env var.
	apiKey := config.ApiKey.ValueString()
	if config.ApiKey.IsNull() || apiKey == "" {
		apiKey = os.Getenv("RACTERMX_API_KEY")
	}

	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing RacterMX API Key",
			"The provider requires an API key to authenticate with the RacterMX API. "+
				"Set the api_key attribute in the provider configuration block or "+
				"set the RACTERMX_API_KEY environment variable.",
		)
		return
	}

	// Resolve base_url: HCL value takes precedence, then default.
	baseURL := config.BaseUrl.ValueString()
	if config.BaseUrl.IsNull() || baseURL == "" {
		baseURL = "https://ractermx.com"
	}

	// Create the API client.
	c := client.NewClient(apiKey, baseURL, p.version)

	// Make the client available to resources and data sources.
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *RactermxProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewDomainResource,
		resources.NewDomainVerificationResource,
		resources.NewAliasResource,
		resources.NewZoneRecordResource,
		resources.NewWebhookResource,
		resources.NewBlocklistEntryResource,
		resources.NewSmtpCredentialResource,
		resources.NewApiKeyResource,
		resources.NewRetentionPolicyResource,
		resources.NewOrganizationResource,
		resources.NewDomainTagResource,
		resources.NewDomainTagAssignmentResource,
		resources.NewAlertRuleResource,
		resources.NewNotificationPreferenceResource,
		resources.NewCheckOverrideResource,
	}
}

func (p *RactermxProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		datasources.NewDomainDnsRecordsDataSource,
		datasources.NewDomainStatisticsDataSource,
		datasources.NewDomainHealthDataSource,
		datasources.NewQuotaDataSource,
		datasources.NewSecurityScoreDataSource,
		datasources.NewSecurityChecksDataSource,
		datasources.NewCheckCatalogDataSource,
		datasources.NewReputationScoreDataSource,
		datasources.NewDmarcComplianceDataSource,
		datasources.NewStatisticsDataSource,
	}
}

// New returns a new provider.Provider for RacterMX.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &RactermxProvider{
			version: version,
		}
	}
}
