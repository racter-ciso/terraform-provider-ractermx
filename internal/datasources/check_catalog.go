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

// Ensure CheckCatalogDataSource satisfies the datasource.DataSource interface.
var _ datasource.DataSource = &CheckCatalogDataSource{}

// CheckCatalogDataSource implements the ractermx_check_catalog data source.
type CheckCatalogDataSource struct {
	client *client.Client
}

// CheckCatalogEntryModel represents a single check catalog entry.
type CheckCatalogEntryModel struct {
	CheckID         types.String `tfsdk:"check_id"`
	Pillar          types.String `tfsdk:"pillar"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	DefaultSeverity types.String `tfsdk:"default_severity"`
	Version         types.String `tfsdk:"version"`
}

// CheckCatalogModel maps the data source schema to a Go struct.
type CheckCatalogModel struct {
	Checks []CheckCatalogEntryModel `tfsdk:"checks"`
}

// NewCheckCatalogDataSource returns a new datasource.DataSource for the check catalog.
func NewCheckCatalogDataSource() datasource.DataSource {
	return &CheckCatalogDataSource{}
}

func (d *CheckCatalogDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_check_catalog"
}

func (d *CheckCatalogDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads the available security check catalog from RacterMX.",
		Attributes: map[string]schema.Attribute{
			"checks": schema.ListNestedAttribute{
				Description: "List of available security checks grouped by pillar.",
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
							Description: "A description of what the check verifies.",
							Computed:    true,
						},
						"default_severity": schema.StringAttribute{
							Description: "The default severity level of the check.",
							Computed:    true,
						},
						"version": schema.StringAttribute{
							Description: "The version of the check.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *CheckCatalogDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *CheckCatalogDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config CheckCatalogModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.Get(ctx, "/check-catalog", false)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Check Catalog",
			"Could not read check catalog: "+err.Error(),
		)
		return
	}

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(result, &envelope); err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Check Catalog Response",
			"Could not parse API response envelope: "+err.Error(),
		)
		return
	}

	config.Checks = []CheckCatalogEntryModel{}

	// Try to parse as an array of checks first.
	var checks []map[string]interface{}
	if err := json.Unmarshal(envelope.Data, &checks); err == nil {
		for _, c := range checks {
			config.Checks = append(config.Checks, parseCheckCatalogEntry(c))
		}
	} else {
		// Try as object with pillar groups.
		var pillarGroups map[string][]map[string]interface{}
		if err := json.Unmarshal(envelope.Data, &pillarGroups); err == nil {
			for pillar, entries := range pillarGroups {
				for _, c := range entries {
					entry := parseCheckCatalogEntry(c)
					if entry.Pillar.IsNull() {
						entry.Pillar = types.StringValue(pillar)
					}
					config.Checks = append(config.Checks, entry)
				}
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

// parseCheckCatalogEntry maps a check catalog entry data object to a CheckCatalogEntryModel.
func parseCheckCatalogEntry(data map[string]interface{}) CheckCatalogEntryModel {
	entry := CheckCatalogEntryModel{}

	if v, ok := data["check_id"].(string); ok {
		entry.CheckID = types.StringValue(v)
	} else {
		entry.CheckID = types.StringNull()
	}
	if v, ok := data["pillar"].(string); ok {
		entry.Pillar = types.StringValue(v)
	} else {
		entry.Pillar = types.StringNull()
	}
	if v, ok := data["name"].(string); ok {
		entry.Name = types.StringValue(v)
	} else {
		entry.Name = types.StringNull()
	}
	if v, ok := data["description"].(string); ok {
		entry.Description = types.StringValue(v)
	} else {
		entry.Description = types.StringNull()
	}
	if v, ok := data["default_severity"].(string); ok {
		entry.DefaultSeverity = types.StringValue(v)
	} else {
		entry.DefaultSeverity = types.StringNull()
	}
	if v, ok := data["version"].(string); ok {
		entry.Version = types.StringValue(v)
	} else {
		entry.Version = types.StringNull()
	}

	return entry
}
