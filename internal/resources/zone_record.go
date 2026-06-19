// Copyright (c) RacterMX
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-ractermx/internal/client"
)

// Ensure ZoneRecordResource satisfies the resource interfaces.
var (
	_ resource.Resource                = &ZoneRecordResource{}
	_ resource.ResourceWithImportState = &ZoneRecordResource{}
)

// ZoneRecordResource implements the ractermx_zone_record resource.
type ZoneRecordResource struct {
	client *client.Client
}

// ZoneRecordResourceModel maps the resource schema to a Go struct.
type ZoneRecordResourceModel struct {
	ID       types.String `tfsdk:"id"`
	DomainID types.Int64  `tfsdk:"domain_id"`
	Name     types.String `tfsdk:"name"`
	Type     types.String `tfsdk:"type"`
	Content  types.String `tfsdk:"content"`
	TTL      types.Int64  `tfsdk:"ttl"`
	Priority types.Int64  `tfsdk:"priority"`
	Weight   types.Int64  `tfsdk:"weight"`
	Port     types.Int64  `tfsdk:"port"`
}

// NewZoneRecordResource returns a new resource.Resource for the zone record resource.
func NewZoneRecordResource() resource.Resource {
	return &ZoneRecordResource{}
}

func (r *ZoneRecordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zone_record"
}

func (r *ZoneRecordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a DNS zone record for a RacterMX-hosted domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The composite ID of the zone record in the format {domain_id}/{name}/{type}/{content}.",
				Computed:    true,
			},
			"domain_id": schema.Int64Attribute{
				Description: "The numeric ID of the domain. Changing this forces a new resource.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The record name (e.g., 'www', '@', 'mail.example.com').",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "The record type (A, AAAA, CNAME, MX, TXT, SRV, NS, CAA, etc.).",
				Required:    true,
			},
			"content": schema.StringAttribute{
				Description: "The record content/value.",
				Required:    true,
			},
			"ttl": schema.Int64Attribute{
				Description: "TTL in seconds (60-86400).",
				Required:    true,
			},
			"priority": schema.Int64Attribute{
				Description: "Priority (0-65535, for MX/SRV records).",
				Optional:    true,
			},
			"weight": schema.Int64Attribute{
				Description: "Weight (0-65535, for SRV records).",
				Optional:    true,
			},
			"port": schema.Int64Attribute{
				Description: "Port (1-65535, for SRV records).",
				Optional:    true,
			},
		},
	}
}

func (r *ZoneRecordResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *ZoneRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ZoneRecordResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := strconv.FormatInt(plan.DomainID.ValueInt64(), 10)

	body := map[string]interface{}{
		"name":    plan.Name.ValueString(),
		"type":    plan.Type.ValueString(),
		"content": plan.Content.ValueString(),
		"ttl":     plan.TTL.ValueInt64(),
	}

	if !plan.Priority.IsNull() && !plan.Priority.IsUnknown() {
		body["priority"] = plan.Priority.ValueInt64()
	}
	if !plan.Weight.IsNull() && !plan.Weight.IsUnknown() {
		body["weight"] = plan.Weight.ValueInt64()
	}
	if !plan.Port.IsNull() && !plan.Port.IsUnknown() {
		body["port"] = plan.Port.ValueInt64()
	}

	_, err := r.client.Post(ctx, "/domains/"+domainID+"/zone-records", body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Zone Record",
			"Could not create zone record: "+err.Error(),
		)
		return
	}

	// Set the composite ID and state from the plan (the API may not return the record).
	plan.ID = types.StringValue(FormatZoneRecordID(
		plan.DomainID.ValueInt64(),
		plan.Name.ValueString(),
		plan.Type.ValueString(),
		plan.Content.ValueString(),
	))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ZoneRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ZoneRecordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := strconv.FormatInt(state.DomainID.ValueInt64(), 10)

	// Zone records have no individual GET endpoint. List all and match.
	result, err := r.client.Get(ctx, "/domains/"+domainID+"/zone-records", true)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Zone Record",
			"Could not read zone records for domain "+domainID+": "+err.Error(),
		)
		return
	}

	// nil result means 404 on the domain — resource was deleted out-of-band.
	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	record, err := findZoneRecord(result, state.Name.ValueString(), state.Type.ValueString(), state.Content.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Zone Records",
			"Could not parse zone records response: "+err.Error(),
		)
		return
	}

	if record == nil {
		// Record not found — removed out-of-band.
		resp.State.RemoveResource(ctx)
		return
	}

	// Map the found record to state.
	var refreshed ZoneRecordResourceModel
	mapZoneRecordToModel(record, state.DomainID.ValueInt64(), &refreshed)

	resp.Diagnostics.Append(resp.State.Set(ctx, &refreshed)...)
}

func (r *ZoneRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ZoneRecordResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ZoneRecordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := strconv.FormatInt(state.DomainID.ValueInt64(), 10)

	// The API uses an old/new pattern to identify and update the record.
	body := map[string]interface{}{
		"old_name":    state.Name.ValueString(),
		"old_type":    state.Type.ValueString(),
		"old_content": state.Content.ValueString(),
		"old_ttl":     state.TTL.ValueInt64(),
		"new_name":    plan.Name.ValueString(),
		"new_type":    plan.Type.ValueString(),
		"new_content": plan.Content.ValueString(),
		"new_ttl":     plan.TTL.ValueInt64(),
	}

	if !plan.Priority.IsNull() && !plan.Priority.IsUnknown() {
		body["new_priority"] = plan.Priority.ValueInt64()
	}
	if !plan.Weight.IsNull() && !plan.Weight.IsUnknown() {
		body["new_weight"] = plan.Weight.ValueInt64()
	}
	if !plan.Port.IsNull() && !plan.Port.IsUnknown() {
		body["new_port"] = plan.Port.ValueInt64()
	}

	_, err := r.client.Patch(ctx, "/domains/"+domainID+"/zone-records", body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Zone Record",
			"Could not update zone record: "+err.Error(),
		)
		return
	}

	// Update the composite ID to reflect the new values.
	plan.ID = types.StringValue(FormatZoneRecordID(
		plan.DomainID.ValueInt64(),
		plan.Name.ValueString(),
		plan.Type.ValueString(),
		plan.Content.ValueString(),
	))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ZoneRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ZoneRecordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := strconv.FormatInt(state.DomainID.ValueInt64(), 10)

	// Delete sends a JSON body identifying the record.
	body := map[string]interface{}{
		"name":    state.Name.ValueString(),
		"type":    state.Type.ValueString(),
		"content": state.Content.ValueString(),
	}

	err := r.client.DeleteWithBody(ctx, "/domains/"+domainID+"/zone-records", body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Zone Record",
			"Could not delete zone record: "+err.Error(),
		)
		return
	}
}

func (r *ZoneRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	domainID, name, recordType, content, err := ParseZoneRecordID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Could not parse zone record import ID: %s", err.Error()),
		)
		return
	}

	state := ZoneRecordResourceModel{
		ID:       types.StringValue(FormatZoneRecordID(domainID, name, recordType, content)),
		DomainID: types.Int64Value(domainID),
		Name:     types.StringValue(name),
		Type:     types.StringValue(recordType),
		Content:  types.StringValue(content),
		TTL:      types.Int64Unknown(),
		Priority: types.Int64Null(),
		Weight:   types.Int64Null(),
		Port:     types.Int64Null(),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// findZoneRecord searches the zone records response for a record matching name, type, and content.
// Returns nil if no match is found.
func findZoneRecord(body []byte, name, recordType, content string) (map[string]interface{}, error) {
	var envelope struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("could not parse zone records response: %w", err)
	}

	for _, raw := range envelope.Data {
		var record map[string]interface{}
		if err := json.Unmarshal(raw, &record); err != nil {
			continue
		}

		rName, _ := record["name"].(string)
		rType, _ := record["type"].(string)
		rContent, _ := record["content"].(string)

		if rName == name && rType == recordType && rContent == content {
			return record, nil
		}
	}

	return nil, nil
}

// mapZoneRecordToModel maps a zone record API response object to a ZoneRecordResourceModel.
func mapZoneRecordToModel(record map[string]interface{}, domainID int64, model *ZoneRecordResourceModel) {
	model.DomainID = types.Int64Value(domainID)

	name, _ := record["name"].(string)
	model.Name = types.StringValue(name)

	recordType, _ := record["type"].(string)
	model.Type = types.StringValue(recordType)

	content, _ := record["content"].(string)
	model.Content = types.StringValue(content)

	if v, ok := record["ttl"].(float64); ok {
		model.TTL = types.Int64Value(int64(v))
	}

	if v, ok := record["priority"].(float64); ok {
		model.Priority = types.Int64Value(int64(v))
	} else {
		model.Priority = types.Int64Null()
	}

	if v, ok := record["weight"].(float64); ok {
		model.Weight = types.Int64Value(int64(v))
	} else {
		model.Weight = types.Int64Null()
	}

	if v, ok := record["port"].(float64); ok {
		model.Port = types.Int64Value(int64(v))
	} else {
		model.Port = types.Int64Null()
	}

	model.ID = types.StringValue(FormatZoneRecordID(domainID, name, recordType, content))
}
