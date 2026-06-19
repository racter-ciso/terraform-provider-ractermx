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

var _ datasource.DataSource = &ReputationScoreDataSource{}

type ReputationScoreDataSource struct {
	client *client.Client
}

type ReputationScoreModel struct {
	DomainID         types.Int64  `tfsdk:"domain_id"`
	CompositeScore   types.Int64  `tfsdk:"composite_score"`
	Grade            types.String `tfsdk:"grade"`
	IsDegraded       types.Bool   `tfsdk:"is_degraded"`
	InsufficientData types.Bool   `tfsdk:"insufficient_data"`
	TotalSent        types.Int64  `tfsdk:"total_sent"`
	ComputedAt       types.String `tfsdk:"computed_at"`
}

func NewReputationScoreDataSource() datasource.DataSource {
	return &ReputationScoreDataSource{}
}

func (d *ReputationScoreDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_reputation_score"
}

func (d *ReputationScoreDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads the outbound email reputation score for a RacterMX domain.",
		Attributes: map[string]schema.Attribute{
			"domain_id": schema.Int64Attribute{
				Description: "The numeric ID of the domain.",
				Required:    true,
			},
			"composite_score": schema.Int64Attribute{
				Description: "The composite reputation score (0-100).",
				Computed:    true,
			},
			"grade": schema.StringAttribute{
				Description: "The reputation grade (A-F).",
				Computed:    true,
			},
			"is_degraded": schema.BoolAttribute{
				Description: "Whether the reputation is currently degraded.",
				Computed:    true,
			},
			"insufficient_data": schema.BoolAttribute{
				Description: "True if there is not enough outbound activity to compute a score.",
				Computed:    true,
			},
			"total_sent": schema.Int64Attribute{
				Description: "Total emails sent in the scoring window.",
				Computed:    true,
			},
			"computed_at": schema.StringAttribute{
				Description: "ISO 8601 timestamp of when the score was computed.",
				Computed:    true,
			},
		},
	}
}

func (d *ReputationScoreDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ReputationScoreDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ReputationScoreModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := config.DomainID.ValueInt64()
	result, err := d.client.Get(ctx, fmt.Sprintf("/domains/%d/reputation", domainID), false)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Reputation Score",
			fmt.Sprintf("Could not read reputation score for domain ID %d: %s", domainID, err.Error()),
		)
		return
	}

	var envelope struct {
		Data json.RawMessage `json:"data"`
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

	if v, ok := data["composite_score"].(float64); ok {
		config.CompositeScore = types.Int64Value(int64(v))
	} else {
		config.CompositeScore = types.Int64Null()
	}
	if v, ok := data["grade"].(string); ok {
		config.Grade = types.StringValue(v)
	} else {
		config.Grade = types.StringNull()
	}
	if v, ok := data["is_degraded"].(bool); ok {
		config.IsDegraded = types.BoolValue(v)
	} else {
		config.IsDegraded = types.BoolValue(false)
	}
	if v, ok := data["insufficient_data"].(bool); ok {
		config.InsufficientData = types.BoolValue(v)
	} else {
		config.InsufficientData = types.BoolValue(false)
	}
	if v, ok := data["total_sent"].(float64); ok {
		config.TotalSent = types.Int64Value(int64(v))
	} else {
		config.TotalSent = types.Int64Value(0)
	}
	if v, ok := data["computed_at"].(string); ok {
		config.ComputedAt = types.StringValue(v)
	} else {
		config.ComputedAt = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
