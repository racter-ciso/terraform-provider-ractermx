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

// Ensure DomainHealthDataSource satisfies the datasource.DataSource interface.
var _ datasource.DataSource = &DomainHealthDataSource{}

// DomainHealthDataSource implements the ractermx_domain_health data source.
type DomainHealthDataSource struct {
	client *client.Client
}

// HealthCheckModel represents a single health check entry (mx, spf, dkim, dmarc).
type HealthCheckModel struct {
	Status    types.String `tfsdk:"status"`
	Message   types.String `tfsdk:"message"`
	CheckedAt types.String `tfsdk:"checked_at"`
}

// HealthChecksModel represents the checks nested object.
type HealthChecksModel struct {
	MX    *HealthCheckModel `tfsdk:"mx"`
	SPF   *HealthCheckModel `tfsdk:"spf"`
	DKIM  *HealthCheckModel `tfsdk:"dkim"`
	DMARC *HealthCheckModel `tfsdk:"dmarc"`
}

// DomainHealthModel maps the data source schema to a Go struct.
type DomainHealthModel struct {
	DomainID       types.Int64        `tfsdk:"domain_id"`
	OverallStatus  types.String       `tfsdk:"overall_status"`
	DomainVerified types.Bool         `tfsdk:"domain_verified"`
	Checks         *HealthChecksModel `tfsdk:"checks"`
}

// NewDomainHealthDataSource returns a new datasource.DataSource for domain health.
func NewDomainHealthDataSource() datasource.DataSource {
	return &DomainHealthDataSource{}
}

func (d *DomainHealthDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_health"
}

func (d *DomainHealthDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	healthCheckAttributes := map[string]schema.Attribute{
		"status": schema.StringAttribute{
			Description: "The status of the health check.",
			Computed:    true,
		},
		"message": schema.StringAttribute{
			Description: "A message describing the health check result.",
			Computed:    true,
		},
		"checked_at": schema.StringAttribute{
			Description: "The timestamp when the check was last performed.",
			Computed:    true,
		},
	}

	resp.Schema = schema.Schema{
		Description: "Reads the health status of a RacterMX domain's DNS configuration.",
		Attributes: map[string]schema.Attribute{
			"domain_id": schema.Int64Attribute{
				Description: "The numeric ID of the domain.",
				Required:    true,
			},
			"overall_status": schema.StringAttribute{
				Description: "The overall health status of the domain (pass, warning, or fail).",
				Computed:    true,
			},
			"domain_verified": schema.BoolAttribute{
				Description: "Whether the domain has been verified.",
				Computed:    true,
			},
			"checks": schema.SingleNestedAttribute{
				Description: "Individual DNS health checks.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"mx": schema.SingleNestedAttribute{
						Description: "MX record health check.",
						Computed:    true,
						Attributes:  healthCheckAttributes,
					},
					"spf": schema.SingleNestedAttribute{
						Description: "SPF record health check.",
						Computed:    true,
						Attributes:  healthCheckAttributes,
					},
					"dkim": schema.SingleNestedAttribute{
						Description: "DKIM record health check.",
						Computed:    true,
						Attributes:  healthCheckAttributes,
					},
					"dmarc": schema.SingleNestedAttribute{
						Description: "DMARC record health check.",
						Computed:    true,
						Attributes:  healthCheckAttributes,
					},
				},
			},
		},
	}
}

func (d *DomainHealthDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DomainHealthDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DomainHealthModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := config.DomainID.ValueInt64()
	result, err := d.client.Get(ctx, fmt.Sprintf("/domains/%d/health", domainID), false)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Domain Health",
			fmt.Sprintf("Could not read health for domain ID %d: %s", domainID, err.Error()),
		)
		return
	}

	// Parse the API response envelope: {"data": {...}}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(result, &envelope); err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Health Response",
			"Could not parse API response envelope: "+err.Error(),
		)
		return
	}

	var data map[string]interface{}
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Health Data",
			"Could not parse health data: "+err.Error(),
		)
		return
	}

	// Map top-level fields.
	if v, ok := data["overall_status"].(string); ok {
		config.OverallStatus = types.StringValue(v)
	} else {
		config.OverallStatus = types.StringNull()
	}
	if v, ok := data["domain_verified"].(bool); ok {
		config.DomainVerified = types.BoolValue(v)
	} else {
		config.DomainVerified = types.BoolNull()
	}

	// Map the checks nested object.
	if checksData, ok := data["checks"].(map[string]interface{}); ok {
		config.Checks = &HealthChecksModel{
			MX:    parseHealthCheck(checksData, "mx"),
			SPF:   parseHealthCheck(checksData, "spf"),
			DKIM:  parseHealthCheck(checksData, "dkim"),
			DMARC: parseHealthCheck(checksData, "dmarc"),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

// parseHealthCheck extracts a health check entry from the checks data.
func parseHealthCheck(data map[string]interface{}, key string) *HealthCheckModel {
	check, ok := data[key].(map[string]interface{})
	if !ok {
		return nil
	}

	model := &HealthCheckModel{}

	if v, ok := check["status"].(string); ok {
		model.Status = types.StringValue(v)
	} else {
		model.Status = types.StringNull()
	}
	if v, ok := check["message"].(string); ok {
		model.Message = types.StringValue(v)
	} else {
		model.Message = types.StringNull()
	}
	if v, ok := check["checked_at"].(string); ok {
		model.CheckedAt = types.StringValue(v)
	} else {
		model.CheckedAt = types.StringNull()
	}

	return model
}
