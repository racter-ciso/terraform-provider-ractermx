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

var _ datasource.DataSource = &DmarcComplianceDataSource{}

type DmarcComplianceDataSource struct {
	client *client.Client
}

type DmarcComplianceModel struct {
	DomainID       types.Int64  `tfsdk:"domain_id"`
	ComplianceRate types.String `tfsdk:"compliance_rate"`
	TotalMessages  types.Int64  `tfsdk:"total_messages"`
	PassedMessages types.Int64  `tfsdk:"passed_messages"`
	CurrentPolicy  types.String `tfsdk:"current_policy"`
	Recommendation types.String `tfsdk:"recommendation"`
}

func NewDmarcComplianceDataSource() datasource.DataSource {
	return &DmarcComplianceDataSource{}
}

func (d *DmarcComplianceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dmarc_compliance"
}

func (d *DmarcComplianceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads DMARC compliance data for a RacterMX domain.",
		Attributes: map[string]schema.Attribute{
			"domain_id": schema.Int64Attribute{
				Description: "The numeric ID of the domain.",
				Required:    true,
			},
			"compliance_rate": schema.StringAttribute{
				Description: "The DMARC compliance rate as a percentage string.",
				Computed:    true,
			},
			"total_messages": schema.Int64Attribute{
				Description: "Total messages evaluated.",
				Computed:    true,
			},
			"passed_messages": schema.Int64Attribute{
				Description: "Messages that passed DMARC.",
				Computed:    true,
			},
			"current_policy": schema.StringAttribute{
				Description: "Current DMARC policy (none, quarantine, reject).",
				Computed:    true,
			},
			"recommendation": schema.StringAttribute{
				Description: "Policy recommendation from the advisor.",
				Computed:    true,
			},
		},
	}
}

func (d *DmarcComplianceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DmarcComplianceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DmarcComplianceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := config.DomainID.ValueInt64()
	result, err := d.client.Get(ctx, fmt.Sprintf("/domains/%d/dmarc/compliance", domainID), false)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading DMARC Compliance",
			fmt.Sprintf("Could not read DMARC compliance for domain ID %d: %s", domainID, err.Error()),
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

	// Extract compliance sub-object
	if compliance, ok := data["compliance"].(map[string]interface{}); ok {
		if v, ok := compliance["rate"].(float64); ok {
			config.ComplianceRate = types.StringValue(fmt.Sprintf("%.1f%%", v*100))
		} else if v, ok := compliance["rate"].(string); ok {
			config.ComplianceRate = types.StringValue(v)
		} else {
			config.ComplianceRate = types.StringNull()
		}
		if v, ok := compliance["total_messages"].(float64); ok {
			config.TotalMessages = types.Int64Value(int64(v))
		} else {
			config.TotalMessages = types.Int64Value(0)
		}
		if v, ok := compliance["passed_messages"].(float64); ok {
			config.PassedMessages = types.Int64Value(int64(v))
		} else {
			config.PassedMessages = types.Int64Value(0)
		}
	}

	// Extract recommendation
	if rec, ok := data["recommendation"].(map[string]interface{}); ok {
		if policy, ok := rec["recommended_policy"].(string); ok {
			config.Recommendation = types.StringValue(policy)
		} else {
			config.Recommendation = types.StringNull()
		}
		if policy, ok := rec["current_policy"].(string); ok {
			config.CurrentPolicy = types.StringValue(policy)
		} else {
			config.CurrentPolicy = types.StringNull()
		}
	} else {
		config.Recommendation = types.StringNull()
		config.CurrentPolicy = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
