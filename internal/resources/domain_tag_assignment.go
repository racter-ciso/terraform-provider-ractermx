// Copyright (c) RacterMX
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-ractermx/internal/client"
)

// Ensure DomainTagAssignmentResource satisfies the resource interfaces.
var (
	_ resource.Resource                = &DomainTagAssignmentResource{}
	_ resource.ResourceWithImportState = &DomainTagAssignmentResource{}
)

// DomainTagAssignmentResource implements the ractermx_domain_tag_assignment resource.
type DomainTagAssignmentResource struct {
	client *client.Client
}

// DomainTagAssignmentResourceModel maps the resource schema to a Go struct.
type DomainTagAssignmentResourceModel struct {
	ID       types.String `tfsdk:"id"`
	DomainID types.Int64  `tfsdk:"domain_id"`
	TagID    types.Int64  `tfsdk:"tag_id"`
}

// NewDomainTagAssignmentResource returns a new resource.Resource for the domain tag assignment resource.
func NewDomainTagAssignmentResource() resource.Resource {
	return &DomainTagAssignmentResource{}
}

func (r *DomainTagAssignmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_tag_assignment"
}

func (r *DomainTagAssignmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Assigns a tag to a RacterMX domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The composite ID of the tag assignment ({domain_id}/{tag_id}).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain_id": schema.Int64Attribute{
				Description: "The numeric ID of the domain. Changing this forces a new resource.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"tag_id": schema.Int64Attribute{
				Description: "The numeric ID of the tag. Changing this forces a new resource.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *DomainTagAssignmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DomainTagAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DomainTagAssignmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := plan.DomainID.ValueInt64()
	tagID := plan.TagID.ValueInt64()

	body := map[string]interface{}{
		"tag_ids": []int64{tagID},
	}

	_, err := r.client.Post(ctx, fmt.Sprintf("/domains/%d/tags", domainID), body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Domain Tag Assignment",
			fmt.Sprintf("Could not assign tag %d to domain %d: %s", tagID, domainID, err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(FormatTagAssignmentID(domainID, tagID))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DomainTagAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DomainTagAssignmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := state.DomainID.ValueInt64()
	tagID := state.TagID.ValueInt64()

	// Read the domain and check if the tag is in its tags array.
	result, err := r.client.Get(ctx, fmt.Sprintf("/domains/%d", domainID), true)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Domain Tag Assignment",
			fmt.Sprintf("Could not read domain %d: %s", domainID, err.Error()),
		)
		return
	}

	// nil result means 404 — domain was deleted out-of-band.
	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Parse the domain response and check for the tag.
	found, err := domainHasTag(result, tagID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Domain Response",
			"Could not parse domain response to check tag assignment: "+err.Error(),
		)
		return
	}

	if !found {
		// Tag assignment was removed out-of-band.
		resp.State.RemoveResource(ctx)
		return
	}

	// State is still valid.
	state.ID = types.StringValue(FormatTagAssignmentID(domainID, tagID))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DomainTagAssignmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// No update — both attributes use RequiresReplace().
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Domain tag assignments cannot be updated. Changes force replacement.",
	)
}

func (r *DomainTagAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DomainTagAssignmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := state.DomainID.ValueInt64()
	tagID := state.TagID.ValueInt64()

	err := r.client.Delete(ctx, fmt.Sprintf("/domains/%d/tags/%d", domainID, tagID))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Domain Tag Assignment",
			fmt.Sprintf("Could not remove tag %d from domain %d: %s", tagID, domainID, err.Error()),
		)
		return
	}
}

func (r *DomainTagAssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	domainID, tagID, err := ParseTagAssignmentID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Could not parse tag assignment import ID: %s", err.Error()),
		)
		return
	}

	state := DomainTagAssignmentResourceModel{
		ID:       types.StringValue(FormatTagAssignmentID(domainID, tagID)),
		DomainID: types.Int64Value(domainID),
		TagID:    types.Int64Value(tagID),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// domainHasTag checks if a domain response contains a specific tag ID in its tags array.
func domainHasTag(body []byte, tagID int64) (bool, error) {
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return false, fmt.Errorf("could not parse domain response envelope: %w", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return false, fmt.Errorf("could not parse domain data: %w", err)
	}

	tags, ok := data["tags"].([]interface{})
	if !ok {
		return false, nil
	}

	for _, t := range tags {
		tagObj, ok := t.(map[string]interface{})
		if !ok {
			continue
		}
		if id, ok := tagObj["id"].(float64); ok && int64(id) == tagID {
			return true, nil
		}
	}

	return false, nil
}
