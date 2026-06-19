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

// Ensure DomainDnsRecordsDataSource satisfies the datasource.DataSource interface.
var _ datasource.DataSource = &DomainDnsRecordsDataSource{}

// DomainDnsRecordsDataSource implements the ractermx_domain_dns_records data source.
type DomainDnsRecordsDataSource struct {
	client *client.Client
}

// DnsRecordModel represents a single DNS record entry.
type DnsRecordModel struct {
	Type  types.String `tfsdk:"type"`
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
	TTL   types.Int64  `tfsdk:"ttl"`
}

// DomainDnsRecordsModel maps the data source schema to a Go struct.
type DomainDnsRecordsModel struct {
	DomainID types.Int64     `tfsdk:"domain_id"`
	MX       *DnsRecordModel `tfsdk:"mx"`
	SPF      *DnsRecordModel `tfsdk:"spf"`
	DKIM     *DnsRecordModel `tfsdk:"dkim"`
	DMARC    *DnsRecordModel `tfsdk:"dmarc"`
}

// NewDomainDnsRecordsDataSource returns a new datasource.DataSource for domain DNS records.
func NewDomainDnsRecordsDataSource() datasource.DataSource {
	return &DomainDnsRecordsDataSource{}
}

func (d *DomainDnsRecordsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_dns_records"
}

func (d *DomainDnsRecordsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	dnsRecordAttributes := map[string]schema.Attribute{
		"type": schema.StringAttribute{
			Description: "The DNS record type.",
			Computed:    true,
		},
		"name": schema.StringAttribute{
			Description: "The DNS record name.",
			Computed:    true,
		},
		"value": schema.StringAttribute{
			Description: "The DNS record value.",
			Computed:    true,
		},
		"ttl": schema.Int64Attribute{
			Description: "The DNS record TTL in seconds.",
			Computed:    true,
		},
	}

	resp.Schema = schema.Schema{
		Description: "Reads the required DNS records for a RacterMX domain.",
		Attributes: map[string]schema.Attribute{
			"domain_id": schema.Int64Attribute{
				Description: "The numeric ID of the domain.",
				Required:    true,
			},
			"mx": schema.SingleNestedAttribute{
				Description: "The MX DNS record.",
				Computed:    true,
				Attributes:  dnsRecordAttributes,
			},
			"spf": schema.SingleNestedAttribute{
				Description: "The SPF DNS record.",
				Computed:    true,
				Attributes:  dnsRecordAttributes,
			},
			"dkim": schema.SingleNestedAttribute{
				Description: "The DKIM DNS record.",
				Computed:    true,
				Attributes:  dnsRecordAttributes,
			},
			"dmarc": schema.SingleNestedAttribute{
				Description: "The DMARC DNS record.",
				Computed:    true,
				Attributes:  dnsRecordAttributes,
			},
		},
	}
}

func (d *DomainDnsRecordsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DomainDnsRecordsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DomainDnsRecordsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := config.DomainID.ValueInt64()
	result, err := d.client.Get(ctx, fmt.Sprintf("/domains/%d/dns-records", domainID), false)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Domain DNS Records",
			fmt.Sprintf("Could not read DNS records for domain ID %d: %s", domainID, err.Error()),
		)
		return
	}

	// Parse the API response envelope: {"data": {...}}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(result, &envelope); err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing DNS Records Response",
			"Could not parse API response envelope: "+err.Error(),
		)
		return
	}

	var data map[string]interface{}
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing DNS Records Data",
			"Could not parse DNS records data: "+err.Error(),
		)
		return
	}

	// Map each record type to the model.
	config.MX = parseDnsRecord(data, "mx")
	config.SPF = parseDnsRecord(data, "spf")
	config.DKIM = parseDnsRecord(data, "dkim")
	config.DMARC = parseDnsRecord(data, "dmarc")

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

// parseDnsRecord extracts a DNS record from the API response data.
func parseDnsRecord(data map[string]interface{}, key string) *DnsRecordModel {
	record, ok := data[key].(map[string]interface{})
	if !ok {
		return nil
	}

	model := &DnsRecordModel{}

	if v, ok := record["type"].(string); ok {
		model.Type = types.StringValue(v)
	} else {
		model.Type = types.StringNull()
	}
	if v, ok := record["name"].(string); ok {
		model.Name = types.StringValue(v)
	} else {
		model.Name = types.StringNull()
	}
	if v, ok := record["value"].(string); ok {
		model.Value = types.StringValue(v)
	} else {
		model.Value = types.StringNull()
	}
	if v, ok := record["ttl"].(float64); ok {
		model.TTL = types.Int64Value(int64(v))
	} else {
		model.TTL = types.Int64Null()
	}

	return model
}
