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

var _ datasource.DataSource = &StatisticsDataSource{}

type StatisticsDataSource struct {
	client *client.Client
}

type StatisticsModel struct {
	DateFrom       types.String `tfsdk:"date_from"`
	DateTo         types.String `tfsdk:"date_to"`
	TotalReceived  types.Int64  `tfsdk:"total_received"`
	TotalForwarded types.Int64  `tfsdk:"total_forwarded"`
	TotalBounced   types.Int64  `tfsdk:"total_bounced"`
	TotalDeferred  types.Int64  `tfsdk:"total_deferred"`
	TotalRejected  types.Int64  `tfsdk:"total_rejected"`
	TotalBytes     types.Int64  `tfsdk:"total_bytes"`
}

func NewStatisticsDataSource() datasource.DataSource {
	return &StatisticsDataSource{}
}

func (d *StatisticsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_statistics"
}

func (d *StatisticsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads aggregated email statistics for the account.",
		Attributes: map[string]schema.Attribute{
			"date_from": schema.StringAttribute{
				Description: "Start date (YYYY-MM-DD). Defaults to 30 days ago.",
				Optional:    true,
				Computed:    true,
			},
			"date_to": schema.StringAttribute{
				Description: "End date (YYYY-MM-DD). Defaults to today.",
				Optional:    true,
				Computed:    true,
			},
			"total_received": schema.Int64Attribute{
				Description: "Total emails received.",
				Computed:    true,
			},
			"total_forwarded": schema.Int64Attribute{
				Description: "Total emails forwarded.",
				Computed:    true,
			},
			"total_bounced": schema.Int64Attribute{
				Description: "Total emails bounced.",
				Computed:    true,
			},
			"total_deferred": schema.Int64Attribute{
				Description: "Total emails deferred.",
				Computed:    true,
			},
			"total_rejected": schema.Int64Attribute{
				Description: "Total emails rejected.",
				Computed:    true,
			},
			"total_bytes": schema.Int64Attribute{
				Description: "Total bytes received.",
				Computed:    true,
			},
		},
	}
}

func (d *StatisticsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *StatisticsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config StatisticsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	path := "/statistics"
	qs := ""
	if !config.DateFrom.IsNull() && !config.DateFrom.IsUnknown() {
		qs += "date_from=" + config.DateFrom.ValueString()
	}
	if !config.DateTo.IsNull() && !config.DateTo.IsUnknown() {
		if qs != "" {
			qs += "&"
		}
		qs += "date_to=" + config.DateTo.ValueString()
	}
	if qs != "" {
		path += "?" + qs
	}

	result, err := d.client.Get(ctx, path, false)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Statistics", err.Error())
		return
	}

	var envelope struct {
		Data json.RawMessage `json:"data"`
		Meta json.RawMessage `json:"meta"`
	}
	if err := json.Unmarshal(result, &envelope); err != nil {
		resp.Diagnostics.AddError("Error Parsing Response", err.Error())
		return
	}

	var data map[string]interface{}
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		resp.Diagnostics.AddError("Error Parsing Data", err.Error())
		return
	}

	// Parse meta for date range
	var meta map[string]interface{}
	if envelope.Meta != nil {
		_ = json.Unmarshal(envelope.Meta, &meta)
	}

	if v, ok := data["total_received"].(float64); ok {
		config.TotalReceived = types.Int64Value(int64(v))
	}
	if v, ok := data["total_forwarded"].(float64); ok {
		config.TotalForwarded = types.Int64Value(int64(v))
	}
	if v, ok := data["total_bounced"].(float64); ok {
		config.TotalBounced = types.Int64Value(int64(v))
	}
	if v, ok := data["total_deferred"].(float64); ok {
		config.TotalDeferred = types.Int64Value(int64(v))
	}
	if v, ok := data["total_rejected"].(float64); ok {
		config.TotalRejected = types.Int64Value(int64(v))
	}
	if v, ok := data["total_bytes"].(float64); ok {
		config.TotalBytes = types.Int64Value(int64(v))
	}

	if meta != nil {
		if v, ok := meta["date_from"].(string); ok {
			config.DateFrom = types.StringValue(v)
		}
		if v, ok := meta["date_to"].(string); ok {
			config.DateTo = types.StringValue(v)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
