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

// Ensure DomainStatisticsDataSource satisfies the datasource.DataSource interface.
var _ datasource.DataSource = &DomainStatisticsDataSource{}

// DomainStatisticsDataSource implements the ractermx_domain_statistics data source.
type DomainStatisticsDataSource struct {
	client *client.Client
}

// DomainStatisticsModel maps the data source schema to a Go struct.
type DomainStatisticsModel struct {
	DomainID       types.Int64  `tfsdk:"domain_id"`
	DateFrom       types.String `tfsdk:"date_from"`
	DateTo         types.String `tfsdk:"date_to"`
	TotalReceived  types.Int64  `tfsdk:"total_received"`
	TotalForwarded types.Int64  `tfsdk:"total_forwarded"`
	TotalBounced   types.Int64  `tfsdk:"total_bounced"`
	TotalDeferred  types.Int64  `tfsdk:"total_deferred"`
	TotalRejected  types.Int64  `tfsdk:"total_rejected"`
}

// NewDomainStatisticsDataSource returns a new datasource.DataSource for domain statistics.
func NewDomainStatisticsDataSource() datasource.DataSource {
	return &DomainStatisticsDataSource{}
}

func (d *DomainStatisticsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_statistics"
}

func (d *DomainStatisticsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads email statistics for a RacterMX domain.",
		Attributes: map[string]schema.Attribute{
			"domain_id": schema.Int64Attribute{
				Description: "The numeric ID of the domain.",
				Required:    true,
			},
			"date_from": schema.StringAttribute{
				Description: "Start date for the statistics period (YYYY-MM-DD format).",
				Optional:    true,
			},
			"date_to": schema.StringAttribute{
				Description: "End date for the statistics period (YYYY-MM-DD format).",
				Optional:    true,
			},
			"total_received": schema.Int64Attribute{
				Description: "Total number of emails received.",
				Computed:    true,
			},
			"total_forwarded": schema.Int64Attribute{
				Description: "Total number of emails forwarded.",
				Computed:    true,
			},
			"total_bounced": schema.Int64Attribute{
				Description: "Total number of emails bounced.",
				Computed:    true,
			},
			"total_deferred": schema.Int64Attribute{
				Description: "Total number of emails deferred.",
				Computed:    true,
			},
			"total_rejected": schema.Int64Attribute{
				Description: "Total number of emails rejected.",
				Computed:    true,
			},
		},
	}
}

func (d *DomainStatisticsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DomainStatisticsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DomainStatisticsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := config.DomainID.ValueInt64()

	// Build the API path with optional query parameters.
	path := fmt.Sprintf("/domains/%d/statistics", domainID)
	separator := "?"

	if !config.DateFrom.IsNull() && !config.DateFrom.IsUnknown() {
		path += separator + "date_from=" + config.DateFrom.ValueString()
		separator = "&"
	}
	if !config.DateTo.IsNull() && !config.DateTo.IsUnknown() {
		path += separator + "date_to=" + config.DateTo.ValueString()
	}

	result, err := d.client.Get(ctx, path, false)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Domain Statistics",
			fmt.Sprintf("Could not read statistics for domain ID %d: %s", domainID, err.Error()),
		)
		return
	}

	// Parse the API response envelope: {"data": {...}}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(result, &envelope); err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Statistics Response",
			"Could not parse API response envelope: "+err.Error(),
		)
		return
	}

	var data map[string]interface{}
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Statistics Data",
			"Could not parse statistics data: "+err.Error(),
		)
		return
	}

	// Map API response fields to the model.
	if v, ok := data["total_received"].(float64); ok {
		config.TotalReceived = types.Int64Value(int64(v))
	} else {
		config.TotalReceived = types.Int64Value(0)
	}
	if v, ok := data["total_forwarded"].(float64); ok {
		config.TotalForwarded = types.Int64Value(int64(v))
	} else {
		config.TotalForwarded = types.Int64Value(0)
	}
	if v, ok := data["total_bounced"].(float64); ok {
		config.TotalBounced = types.Int64Value(int64(v))
	} else {
		config.TotalBounced = types.Int64Value(0)
	}
	if v, ok := data["total_deferred"].(float64); ok {
		config.TotalDeferred = types.Int64Value(int64(v))
	} else {
		config.TotalDeferred = types.Int64Value(0)
	}
	if v, ok := data["total_rejected"].(float64); ok {
		config.TotalRejected = types.Int64Value(int64(v))
	} else {
		config.TotalRejected = types.Int64Value(0)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
