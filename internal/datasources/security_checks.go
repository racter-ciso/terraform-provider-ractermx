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

// Ensure SecurityChecksDataSource satisfies the datasource.DataSource interface.
var _ datasource.DataSource = &SecurityChecksDataSource{}

// SecurityChecksDataSource implements the ractermx_security_checks data source.
type SecurityChecksDataSource struct {
	client *client.Client
}

// SecurityFindingModel represents a single security finding.
type SecurityFindingModel struct {
	CheckID      types.String `tfsdk:"check_id"`
	Pillar       types.String `tfsdk:"pillar"`
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	Status       types.String `tfsdk:"status"`
	Severity     types.String `tfsdk:"severity"`
	FixAvailable types.Bool   `tfsdk:"fix_available"`
}

// SecurityChecksModel maps the data source schema to a Go struct.
type SecurityChecksModel struct {
	DomainID types.Int64            `tfsdk:"domain_id"`
	Findings []SecurityFindingModel `tfsdk:"findings"`
}

// NewSecurityChecksDataSource returns a new datasource.DataSource for security checks.
func NewSecurityChecksDataSource() datasource.DataSource {
	return &SecurityChecksDataSource{}
}

func (d *SecurityChecksDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_security_checks"
}

func (d *SecurityChecksDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads security findings for a RacterMX domain.",
		Attributes: map[string]schema.Attribute{
			"domain_id": schema.Int64Attribute{
				Description: "The numeric ID of the domain.",
				Required:    true,
			},
			"findings": schema.ListNestedAttribute{
				Description: "List of security findings grouped by pillar.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"check_id": schema.StringAttribute{
							Description: "The unique identifier of the check.",
							Computed:    true,
						},
						"pillar": schema.StringAttribute{
							Description: "The security pillar (identity, shadow, reputation).",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the check.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "A description of the check.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "The status of the finding (pass, fail, warning).",
							Computed:    true,
						},
						"severity": schema.StringAttribute{
							Description: "The severity level of the finding.",
							Computed:    true,
						},
						"fix_available": schema.BoolAttribute{
							Description: "Whether an automated fix is available.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *SecurityChecksDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SecurityChecksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config SecurityChecksModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := config.DomainID.ValueInt64()
	result, err := d.client.Get(ctx, fmt.Sprintf("/domains/%d/security", domainID), false)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Security Checks",
			fmt.Sprintf("Could not read security checks for domain ID %d: %s", domainID, err.Error()),
		)
		return
	}

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(result, &envelope); err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Security Checks Response",
			"Could not parse API response envelope: "+err.Error(),
		)
		return
	}

	// The data may be an array of findings or an object with pillar groups.
	// Try to parse as an array first, then as a map of pillar groups.
	config.Findings = []SecurityFindingModel{}

	var findings []map[string]interface{}
	if err := json.Unmarshal(envelope.Data, &findings); err == nil {
		// Direct array of findings.
		for _, f := range findings {
			config.Findings = append(config.Findings, parseSecurityFinding(f))
		}
	} else {
		// Try as object with pillar groups.
		var pillarGroups map[string][]map[string]interface{}
		if err := json.Unmarshal(envelope.Data, &pillarGroups); err == nil {
			for pillar, checks := range pillarGroups {
				for _, f := range checks {
					finding := parseSecurityFinding(f)
					if finding.Pillar.IsNull() {
						finding.Pillar = types.StringValue(pillar)
					}
					config.Findings = append(config.Findings, finding)
				}
			}
		} else {
			// Try as a single object with a "findings" or "checks" key.
			var wrapper map[string]json.RawMessage
			if err := json.Unmarshal(envelope.Data, &wrapper); err == nil {
				for key, raw := range wrapper {
					if key == "findings" || key == "checks" {
						var items []map[string]interface{}
						if err := json.Unmarshal(raw, &items); err == nil {
							for _, f := range items {
								config.Findings = append(config.Findings, parseSecurityFinding(f))
							}
						}
					}
				}
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

// parseSecurityFinding maps a finding data object to a SecurityFindingModel.
func parseSecurityFinding(data map[string]interface{}) SecurityFindingModel {
	finding := SecurityFindingModel{}

	if v, ok := data["check_id"].(string); ok {
		finding.CheckID = types.StringValue(v)
	} else {
		finding.CheckID = types.StringNull()
	}
	if v, ok := data["pillar"].(string); ok {
		finding.Pillar = types.StringValue(v)
	} else {
		finding.Pillar = types.StringNull()
	}
	if v, ok := data["name"].(string); ok {
		finding.Name = types.StringValue(v)
	} else {
		finding.Name = types.StringNull()
	}
	if v, ok := data["description"].(string); ok {
		finding.Description = types.StringValue(v)
	} else {
		finding.Description = types.StringNull()
	}
	if v, ok := data["status"].(string); ok {
		finding.Status = types.StringValue(v)
	} else {
		finding.Status = types.StringNull()
	}
	if v, ok := data["severity"].(string); ok {
		finding.Severity = types.StringValue(v)
	} else {
		finding.Severity = types.StringNull()
	}
	if v, ok := data["fix_available"].(bool); ok {
		finding.FixAvailable = types.BoolValue(v)
	} else {
		finding.FixAvailable = types.BoolValue(false)
	}

	return finding
}
