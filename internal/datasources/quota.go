// Copyright (c) RacterMX
// SPDX-License-Identifier: MPL-2.0

package datasources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-ractermx/internal/client"
)

// Ensure QuotaDataSource satisfies the datasource.DataSource interface.
var _ datasource.DataSource = &QuotaDataSource{}

// QuotaDataSource implements the ractermx_quota data source.
type QuotaDataSource struct {
	client *client.Client
}

// QuotaModel maps the data source schema to a Go struct.
type QuotaModel struct {
	DomainsLimit          types.Int64 `tfsdk:"domains_limit"`
	DomainsUsed           types.Int64 `tfsdk:"domains_used"`
	AliasesLimit          types.Int64 `tfsdk:"aliases_limit"`
	AliasesUsed           types.Int64 `tfsdk:"aliases_used"`
	SmtpCredentialsLimit  types.Int64 `tfsdk:"smtp_credentials_limit"`
	SmtpCredentialsUsed   types.Int64 `tfsdk:"smtp_credentials_used"`
}

// NewQuotaDataSource returns a new datasource.DataSource for quota.
func NewQuotaDataSource() datasource.DataSource {
	return &QuotaDataSource{}
}

func (d *QuotaDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_quota"
}

func (d *QuotaDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads account quota information from RacterMX.",
		Attributes: map[string]schema.Attribute{
			"domains_limit": schema.Int64Attribute{
				Description: "Maximum number of domains allowed.",
				Computed:    true,
			},
			"domains_used": schema.Int64Attribute{
				Description: "Number of domains currently in use.",
				Computed:    true,
			},
			"aliases_limit": schema.Int64Attribute{
				Description: "Maximum number of aliases allowed.",
				Computed:    true,
			},
			"aliases_used": schema.Int64Attribute{
				Description: "Number of aliases currently in use.",
				Computed:    true,
			},
			"smtp_credentials_limit": schema.Int64Attribute{
				Description: "Maximum number of SMTP credentials allowed.",
				Computed:    true,
			},
			"smtp_credentials_used": schema.Int64Attribute{
				Description: "Number of SMTP credentials currently in use.",
				Computed:    true,
			},
		},
	}
}

func (d *QuotaDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = c
}

func (d *QuotaDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config QuotaModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.Get(ctx, "/quota", false)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Quota",
			"Could not read quota: "+err.Error(),
		)
		return
	}

	// Parse the API response envelope: {"data": {...}}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(result, &envelope); err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Quota Response",
			"Could not parse API response envelope: "+err.Error(),
		)
		return
	}

	var data map[string]interface{}
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Quota Data",
			"Could not parse quota data: "+err.Error(),
		)
		return
	}

	// Map API response fields to the model.
	if v, ok := data["domains_limit"].(float64); ok {
		config.DomainsLimit = types.Int64Value(int64(v))
	} else {
		config.DomainsLimit = types.Int64Value(0)
	}
	if v, ok := data["domains_used"].(float64); ok {
		config.DomainsUsed = types.Int64Value(int64(v))
	} else {
		config.DomainsUsed = types.Int64Value(0)
	}
	if v, ok := data["aliases_limit"].(float64); ok {
		config.AliasesLimit = types.Int64Value(int64(v))
	} else {
		config.AliasesLimit = types.Int64Value(0)
	}
	if v, ok := data["aliases_used"].(float64); ok {
		config.AliasesUsed = types.Int64Value(int64(v))
	} else {
		config.AliasesUsed = types.Int64Value(0)
	}
	if v, ok := data["smtp_credentials_limit"].(float64); ok {
		config.SmtpCredentialsLimit = types.Int64Value(int64(v))
	} else {
		config.SmtpCredentialsLimit = types.Int64Value(0)
	}
	if v, ok := data["smtp_credentials_used"].(float64); ok {
		config.SmtpCredentialsUsed = types.Int64Value(int64(v))
	} else {
		config.SmtpCredentialsUsed = types.Int64Value(0)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
