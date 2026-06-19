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

// Ensure SecurityScoreDataSource satisfies the datasource.DataSource interface.
var _ datasource.DataSource = &SecurityScoreDataSource{}

// SecurityScoreDataSource implements the ractermx_security_score data source.
type SecurityScoreDataSource struct {
	client *client.Client
}

// SecurityScorePillarModel represents a single pillar score.
type SecurityScorePillarModel struct {
	Name  types.String `tfsdk:"name"`
	Score types.Int64  `tfsdk:"score"`
	Grade types.String `tfsdk:"grade"`
}

// SecurityScoreModel maps the data source schema to a Go struct.
type SecurityScoreModel struct {
	DomainID     types.Int64                `tfsdk:"domain_id"`
	OverallScore types.Int64                `tfsdk:"overall_score"`
	Grade        types.String               `tfsdk:"grade"`
	Pillars      []SecurityScorePillarModel `tfsdk:"pillars"`
}

// NewSecurityScoreDataSource returns a new datasource.DataSource for security score.
func NewSecurityScoreDataSource() datasource.DataSource {
	return &SecurityScoreDataSource{}
}

func (d *SecurityScoreDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_security_score"
}

func (d *SecurityScoreDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads the security posture score for a RacterMX domain.",
		Attributes: map[string]schema.Attribute{
			"domain_id": schema.Int64Attribute{
				Description: "The numeric ID of the domain.",
				Required:    true,
			},
			"overall_score": schema.Int64Attribute{
				Description: "The overall security posture score.",
				Computed:    true,
			},
			"grade": schema.StringAttribute{
				Description: "The overall security grade.",
				Computed:    true,
			},
			"pillars": schema.ListNestedAttribute{
				Description: "Breakdown of scores by security pillar.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The name of the security pillar.",
							Computed:    true,
						},
						"score": schema.Int64Attribute{
							Description: "The score for this pillar.",
							Computed:    true,
						},
						"grade": schema.StringAttribute{
							Description: "The grade for this pillar.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *SecurityScoreDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SecurityScoreDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config SecurityScoreModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := config.DomainID.ValueInt64()
	result, err := d.client.Get(ctx, fmt.Sprintf("/domains/%d/security/score", domainID), false)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Security Score",
			fmt.Sprintf("Could not read security score for domain ID %d: %s", domainID, err.Error()),
		)
		return
	}

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(result, &envelope); err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Security Score Response",
			"Could not parse API response envelope: "+err.Error(),
		)
		return
	}

	var data map[string]interface{}
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Security Score Data",
			"Could not parse security score data: "+err.Error(),
		)
		return
	}

	if v, ok := data["overall_score"].(float64); ok {
		config.OverallScore = types.Int64Value(int64(v))
	} else {
		config.OverallScore = types.Int64Value(0)
	}
	if v, ok := data["grade"].(string); ok {
		config.Grade = types.StringValue(v)
	} else {
		config.Grade = types.StringNull()
	}

	// Parse pillars array.
	config.Pillars = []SecurityScorePillarModel{}
	if pillarsRaw, ok := data["pillars"].([]interface{}); ok {
		for _, pRaw := range pillarsRaw {
			p, ok := pRaw.(map[string]interface{})
			if !ok {
				continue
			}
			pillar := SecurityScorePillarModel{}
			if v, ok := p["name"].(string); ok {
				pillar.Name = types.StringValue(v)
			} else {
				pillar.Name = types.StringNull()
			}
			if v, ok := p["score"].(float64); ok {
				pillar.Score = types.Int64Value(int64(v))
			} else {
				pillar.Score = types.Int64Value(0)
			}
			if v, ok := p["grade"].(string); ok {
				pillar.Grade = types.StringValue(v)
			} else {
				pillar.Grade = types.StringNull()
			}
			config.Pillars = append(config.Pillars, pillar)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
