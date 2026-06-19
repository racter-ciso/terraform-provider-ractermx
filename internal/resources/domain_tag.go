// Copyright (c) RacterMX
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-ractermx/internal/client"
)

// Ensure DomainTagResource satisfies the resource interfaces.
var (
	_ resource.Resource                = &DomainTagResource{}
	_ resource.ResourceWithImportState = &DomainTagResource{}
)

// DomainTagResource implements the ractermx_domain_tag resource.
type DomainTagResource struct {
	client *client.Client
}

// DomainTagResourceModel maps the resource schema to a Go struct.
type DomainTagResourceModel struct {
	ID           types.Int64  `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Color        types.String `tfsdk:"color"`
	DomainsCount types.Int64  `tfsdk:"domains_count"`
}

// NewDomainTagResource returns a new resource.Resource for the domain tag resource.
func NewDomainTagResource() resource.Resource {
	return &DomainTagResource{}
}

func (r *DomainTagResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_tag"
}

func (r *DomainTagResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a RacterMX domain tag.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The numeric ID of the tag.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the tag (max 50 chars).",
				Required:    true,
			},
			"color": schema.StringAttribute{
				Description: "The hex color of the tag (e.g., #3b82f6).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("#3b82f6"),
			},
			"domains_count": schema.Int64Attribute{
				Description: "The number of domains assigned to this tag.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *DomainTagResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DomainTagResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DomainTagResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"name": plan.Name.ValueString(),
	}
	if !plan.Color.IsNull() && !plan.Color.IsUnknown() {
		body["color"] = plan.Color.ValueString()
	}

	result, err := r.client.Post(ctx, "/tags", body)
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.StatusCode == 409 {
			resp.Diagnostics.AddError(
				"Domain Tag Already Exists",
				fmt.Sprintf("A tag with name '%s' already exists: %s", plan.Name.ValueString(), apiErr.Message),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error Creating Domain Tag",
			"Could not create domain tag, unexpected error: "+err.Error(),
		)
		return
	}

	var state DomainTagResourceModel
	if diags := parseDomainTagResponse(result, &state); diags != nil {
		resp.Diagnostics.AddError(diags.Summary, diags.Detail)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DomainTagResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DomainTagResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tagID := state.ID.ValueInt64()

	// Tags have no individual GET endpoint. List all and match by ID.
	result, err := r.client.Get(ctx, "/tags", false)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Domain Tag",
			"Could not read domain tags: "+err.Error(),
		)
		return
	}

	tag, err := findDomainTagByID(result, tagID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Domain Tags",
			"Could not parse domain tags response: "+err.Error(),
		)
		return
	}

	if tag == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	var refreshed DomainTagResourceModel
	parseDomainTagData(tag, &refreshed)

	resp.Diagnostics.Append(resp.State.Set(ctx, &refreshed)...)
}

func (r *DomainTagResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DomainTagResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state DomainTagResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tagID := strconv.FormatInt(state.ID.ValueInt64(), 10)

	body := map[string]interface{}{
		"name":  plan.Name.ValueString(),
		"color": plan.Color.ValueString(),
	}

	result, err := r.client.Patch(ctx, "/tags/"+tagID, body)
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.StatusCode == 409 {
			resp.Diagnostics.AddError(
				"Domain Tag Already Exists",
				fmt.Sprintf("A tag with name '%s' already exists: %s", plan.Name.ValueString(), apiErr.Message),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error Updating Domain Tag",
			"Could not update domain tag ID "+tagID+": "+err.Error(),
		)
		return
	}

	var updated DomainTagResourceModel
	if diags := parseDomainTagResponse(result, &updated); diags != nil {
		resp.Diagnostics.AddError(diags.Summary, diags.Detail)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
}

func (r *DomainTagResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DomainTagResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tagID := strconv.FormatInt(state.ID.ValueInt64(), 10)

	err := r.client.Delete(ctx, "/tags/"+tagID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Domain Tag",
			"Could not delete domain tag ID "+tagID+": "+err.Error(),
		)
		return
	}
}

func (r *DomainTagResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected a numeric tag ID, got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

// domainTagDiag is a simple struct for returning parse errors.
type domainTagDiag struct {
	Summary string
	Detail  string
}

// findDomainTagByID searches the tags list response for a tag matching the given ID.
func findDomainTagByID(body []byte, tagID int64) (map[string]interface{}, error) {
	var envelope struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("could not parse tags response: %w", err)
	}

	for _, raw := range envelope.Data {
		var tag map[string]interface{}
		if err := json.Unmarshal(raw, &tag); err != nil {
			continue
		}

		if id, ok := tag["id"].(float64); ok && int64(id) == tagID {
			return tag, nil
		}
	}

	return nil, nil
}

// parseDomainTagResponse parses the API response envelope {"data": {...}} and maps to a DomainTagResourceModel.
func parseDomainTagResponse(body []byte, model *DomainTagResourceModel) *domainTagDiag {
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return &domainTagDiag{
			Summary: "Error Parsing Domain Tag Response",
			Detail:  "Could not parse API response envelope: " + err.Error(),
		}
	}

	var data map[string]interface{}
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return &domainTagDiag{
			Summary: "Error Parsing Domain Tag Data",
			Detail:  "Could not parse domain tag data: " + err.Error(),
		}
	}

	parseDomainTagData(data, model)
	return nil
}

// parseDomainTagData maps a domain tag data object to a DomainTagResourceModel.
func parseDomainTagData(data map[string]interface{}, model *DomainTagResourceModel) {
	if v, ok := data["id"].(float64); ok {
		model.ID = types.Int64Value(int64(v))
	}
	if v, ok := data["name"].(string); ok {
		model.Name = types.StringValue(v)
	}
	if v, ok := data["color"].(string); ok {
		model.Color = types.StringValue(v)
	} else {
		model.Color = types.StringValue("#3b82f6")
	}
	if v, ok := data["domains_count"].(float64); ok {
		model.DomainsCount = types.Int64Value(int64(v))
	} else {
		model.DomainsCount = types.Int64Value(0)
	}
}
