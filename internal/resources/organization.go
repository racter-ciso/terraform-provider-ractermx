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
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-ractermx/internal/client"
)

// Ensure OrganizationResource satisfies the resource interfaces.
var (
	_ resource.Resource                = &OrganizationResource{}
	_ resource.ResourceWithImportState = &OrganizationResource{}
)

// OrganizationResource implements the ractermx_organization resource.
type OrganizationResource struct {
	client *client.Client
}

// OrganizationResourceModel maps the resource schema to a Go struct.
type OrganizationResourceModel struct {
	ID                types.Int64  `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	ParentID          types.Int64  `tfsdk:"parent_id"`
	// Computed
	UsersCount        types.Int64  `tfsdk:"users_count"`
	DomainsCount      types.Int64  `tfsdk:"domains_count"`
	TotalDomainsCount types.Int64  `tfsdk:"total_domains_count"`
}

// NewOrganizationResource returns a new resource.Resource for the organization resource.
func NewOrganizationResource() resource.Resource {
	return &OrganizationResource{}
}

func (r *OrganizationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (r *OrganizationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a RacterMX organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The numeric ID of the organization.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the organization.",
				Required:    true,
			},
			"parent_id": schema.Int64Attribute{
				Description: "The numeric ID of the parent organization.",
				Required:    true,
			},
			// Computed attributes
			"users_count": schema.Int64Attribute{
				Description: "The number of users in this organization.",
				Computed:    true,
			},
			"domains_count": schema.Int64Attribute{
				Description: "The number of domains directly in this organization.",
				Computed:    true,
			},
			"total_domains_count": schema.Int64Attribute{
				Description: "The total number of domains including child organizations.",
				Computed:    true,
			},
		},
	}
}

func (r *OrganizationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrganizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrganizationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"name":      plan.Name.ValueString(),
		"parent_id": plan.ParentID.ValueInt64(),
	}

	result, err := r.client.Post(ctx, "/organizations", body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Organization",
			"Could not create organization, unexpected error: "+err.Error(),
		)
		return
	}

	var state OrganizationResourceModel
	if diags := parseOrganizationResponse(result, &state); diags != nil {
		resp.Diagnostics.AddError(diags.Summary, diags.Detail)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OrganizationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrganizationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.ID.ValueInt64()

	// The organizations API returns a tree structure. We need to list all and traverse to find by ID.
	result, err := r.client.Get(ctx, "/organizations", true)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Organization",
			fmt.Sprintf("Could not read organizations: %s", err.Error()),
		)
		return
	}

	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	org, err := findOrganizationByID(result, orgID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Organizations",
			"Could not parse organizations response: "+err.Error(),
		)
		return
	}

	if org == nil {
		// Organization not found — removed out-of-band.
		resp.State.RemoveResource(ctx)
		return
	}

	var refreshed OrganizationResourceModel
	parseOrganizationData(org, &refreshed)

	resp.Diagnostics.Append(resp.State.Set(ctx, &refreshed)...)
}

func (r *OrganizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrganizationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state OrganizationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := strconv.FormatInt(state.ID.ValueInt64(), 10)

	body := map[string]interface{}{}

	if !plan.Name.Equal(state.Name) {
		body["name"] = plan.Name.ValueString()
	}
	if !plan.ParentID.Equal(state.ParentID) {
		body["parent_id"] = plan.ParentID.ValueInt64()
	}

	result, err := r.client.Patch(ctx, "/organizations/"+orgID, body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Organization",
			"Could not update organization ID "+orgID+": "+err.Error(),
		)
		return
	}

	var updated OrganizationResourceModel
	if diags := parseOrganizationResponse(result, &updated); diags != nil {
		resp.Diagnostics.AddError(diags.Summary, diags.Detail)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
}

func (r *OrganizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrganizationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := strconv.FormatInt(state.ID.ValueInt64(), 10)

	err := r.client.Delete(ctx, "/organizations/"+orgID)
	if err != nil {
		// Provide clear error messages for precondition failures and primary org protection.
		resp.Diagnostics.AddError(
			"Error Deleting Organization",
			"Could not delete organization ID "+orgID+": "+err.Error(),
		)
		return
	}
}

func (r *OrganizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected a numeric organization ID, got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

// organizationDiag is a simple struct for returning parse errors.
type organizationDiag struct {
	Summary string
	Detail  string
}

// findOrganizationByID searches the organizations tree response for an org matching the given ID.
// The API returns {"data": [...]} where each org may have a "children" array.
func findOrganizationByID(body []byte, orgID int64) (map[string]interface{}, error) {
	var envelope struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("could not parse organizations response: %w", err)
	}

	for _, raw := range envelope.Data {
		var org map[string]interface{}
		if err := json.Unmarshal(raw, &org); err != nil {
			continue
		}

		if found := searchOrgTree(org, orgID); found != nil {
			return found, nil
		}
	}

	return nil, nil
}

// searchOrgTree recursively searches an organization tree node for the given ID.
func searchOrgTree(org map[string]interface{}, orgID int64) map[string]interface{} {
	if id, ok := org["id"].(float64); ok && int64(id) == orgID {
		return org
	}

	// Recursively search children.
	if childrenRaw, ok := org["children"].([]interface{}); ok {
		for _, childRaw := range childrenRaw {
			if child, ok := childRaw.(map[string]interface{}); ok {
				if found := searchOrgTree(child, orgID); found != nil {
					return found
				}
			}
		}
	}

	return nil
}

// parseOrganizationResponse parses the API response envelope {"data": {...}} and maps to an OrganizationResourceModel.
func parseOrganizationResponse(body []byte, model *OrganizationResourceModel) *organizationDiag {
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return &organizationDiag{
			Summary: "Error Parsing Organization Response",
			Detail:  "Could not parse API response envelope: " + err.Error(),
		}
	}

	var data map[string]interface{}
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return &organizationDiag{
			Summary: "Error Parsing Organization Data",
			Detail:  "Could not parse organization data: " + err.Error(),
		}
	}

	parseOrganizationData(data, model)
	return nil
}

// parseOrganizationData maps an organization data object to an OrganizationResourceModel.
func parseOrganizationData(data map[string]interface{}, model *OrganizationResourceModel) {
	if v, ok := data["id"].(float64); ok {
		model.ID = types.Int64Value(int64(v))
	}
	if v, ok := data["name"].(string); ok {
		model.Name = types.StringValue(v)
	}
	if v, ok := data["parent_id"].(float64); ok {
		model.ParentID = types.Int64Value(int64(v))
	} else {
		model.ParentID = types.Int64Null()
	}
	if v, ok := data["users_count"].(float64); ok {
		model.UsersCount = types.Int64Value(int64(v))
	} else {
		model.UsersCount = types.Int64Value(0)
	}
	if v, ok := data["domains_count"].(float64); ok {
		model.DomainsCount = types.Int64Value(int64(v))
	} else {
		model.DomainsCount = types.Int64Value(0)
	}
	if v, ok := data["total_domains_count"].(float64); ok {
		model.TotalDomainsCount = types.Int64Value(int64(v))
	} else {
		model.TotalDomainsCount = types.Int64Value(0)
	}
}
