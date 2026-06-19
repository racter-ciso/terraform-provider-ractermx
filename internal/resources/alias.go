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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-ractermx/internal/client"
)

// Ensure AliasResource satisfies the resource interfaces.
var (
	_ resource.Resource                = &AliasResource{}
	_ resource.ResourceWithImportState = &AliasResource{}
)

// AliasResource implements the ractermx_alias resource.
type AliasResource struct {
	client *client.Client
}

// AliasResourceModel maps the resource schema to a Go struct.
type AliasResourceModel struct {
	ID         types.Int64  `tfsdk:"id"`
	DomainID   types.Int64  `tfsdk:"domain_id"`
	LocalPart  types.String `tfsdk:"local_part"`
	ForwardTo  types.String `tfsdk:"forward_to"`
	IsCatchall types.Bool   `tfsdk:"is_catchall"`
	Description types.String `tfsdk:"description"`
	// Computed
	IsActive   types.Bool   `tfsdk:"is_active"`
	IsWildcard types.Bool   `tfsdk:"is_wildcard"`
	CreatedAt  types.String `tfsdk:"created_at"`
	UpdatedAt  types.String `tfsdk:"updated_at"`
}

// NewAliasResource returns a new resource.Resource for the alias resource.
func NewAliasResource() resource.Resource {
	return &AliasResource{}
}

func (r *AliasResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alias"
}

func (r *AliasResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a RacterMX email alias.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The numeric ID of the alias.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"domain_id": schema.Int64Attribute{
				Description: "The numeric ID of the domain this alias belongs to. Changing this forces a new resource.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"local_part": schema.StringAttribute{
				Description: "The local part of the alias (e.g., 'info' for info@domain.com). Changing this forces a new resource. When is_catchall is true, this is set to '*'.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"forward_to": schema.StringAttribute{
				Description: "The email address to forward mail to.",
				Required:    true,
			},
			"is_catchall": schema.BoolAttribute{
				Description: "Whether this is a catch-all alias. When true, local_part is set to '*'.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Description: "A description for this alias.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			// Computed attributes
			"is_active": schema.BoolAttribute{
				Description: "Whether the alias is active.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"is_wildcard": schema.BoolAttribute{
				Description: "Whether the alias is a wildcard alias.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the alias was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the alias was last updated.",
				Computed:    true,
			},
		},
	}
}

func (r *AliasResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AliasResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AliasResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := strconv.FormatInt(plan.DomainID.ValueInt64(), 10)

	// Build request body from the plan.
	body := map[string]interface{}{
		"forward_to": plan.ForwardTo.ValueString(),
	}

	// When is_catchall is true, set local_part to "*".
	if !plan.IsCatchall.IsNull() && !plan.IsCatchall.IsUnknown() && plan.IsCatchall.ValueBool() {
		body["is_catchall"] = true
		body["local_part"] = "*"
	} else if !plan.LocalPart.IsNull() && !plan.LocalPart.IsUnknown() {
		body["local_part"] = plan.LocalPart.ValueString()
	}

	if !plan.IsCatchall.IsNull() && !plan.IsCatchall.IsUnknown() {
		body["is_catchall"] = plan.IsCatchall.ValueBool()
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		body["description"] = plan.Description.ValueString()
	}

	result, err := r.client.Post(ctx, "/domains/"+domainID+"/aliases", body)
	if err != nil {
		// Check for 409 Conflict and provide a clear error message with the alias address.
		if apiErr, ok := err.(*client.APIError); ok && apiErr.StatusCode == 409 {
			localPart := plan.LocalPart.ValueString()
			if plan.IsCatchall.ValueBool() {
				localPart = "*"
			}
			resp.Diagnostics.AddError(
				"Alias Already Exists",
				fmt.Sprintf("An alias for '%s' already exists on domain %s: %s", localPart, domainID, apiErr.Message),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error Creating Alias",
			"Could not create alias, unexpected error: "+err.Error(),
		)
		return
	}

	// Parse the API response envelope: {"data": {...}}
	var state AliasResourceModel
	if diags := parseAliasResponse(result, &state); diags != nil {
		resp.Diagnostics.AddError(diags.Summary, diags.Detail)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AliasResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AliasResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	aliasID := strconv.FormatInt(state.ID.ValueInt64(), 10)
	result, err := r.client.Get(ctx, "/aliases/"+aliasID, true)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Alias",
			"Could not read alias ID "+aliasID+": "+err.Error(),
		)
		return
	}

	// nil result means 404 — resource was deleted out-of-band.
	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	var refreshed AliasResourceModel
	if diags := parseAliasResponse(result, &refreshed); diags != nil {
		resp.Diagnostics.AddError(diags.Summary, diags.Detail)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &refreshed)...)
}

func (r *AliasResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AliasResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state AliasResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	aliasID := strconv.FormatInt(state.ID.ValueInt64(), 10)

	// Only forward_to, description, and is_active can be updated.
	body := map[string]interface{}{}

	if !plan.ForwardTo.Equal(state.ForwardTo) {
		body["forward_to"] = plan.ForwardTo.ValueString()
	}
	if !plan.Description.Equal(state.Description) {
		if plan.Description.IsNull() {
			body["description"] = nil
		} else {
			body["description"] = plan.Description.ValueString()
		}
	}
	if !plan.IsActive.Equal(state.IsActive) {
		body["is_active"] = plan.IsActive.ValueBool()
	}

	result, err := r.client.Patch(ctx, "/aliases/"+aliasID, body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Alias",
			"Could not update alias ID "+aliasID+": "+err.Error(),
		)
		return
	}

	var updated AliasResourceModel
	if diags := parseAliasResponse(result, &updated); diags != nil {
		resp.Diagnostics.AddError(diags.Summary, diags.Detail)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
}

func (r *AliasResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AliasResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	aliasID := strconv.FormatInt(state.ID.ValueInt64(), 10)

	err := r.client.Delete(ctx, "/aliases/"+aliasID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Alias",
			"Could not delete alias ID "+aliasID+": "+err.Error(),
		)
		return
	}
}

func (r *AliasResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID as an integer.
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected a numeric alias ID, got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

// aliasDiag is a simple struct for returning parse errors.
type aliasDiag struct {
	Summary string
	Detail  string
}

// parseAliasResponse parses the API response envelope and maps the data to an AliasResourceModel.
func parseAliasResponse(body []byte, model *AliasResourceModel) *aliasDiag {
	// Parse the outer envelope: {"data": {...}}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return &aliasDiag{
			Summary: "Error Parsing Alias Response",
			Detail:  "Could not parse API response envelope: " + err.Error(),
		}
	}

	// Parse the inner data object.
	var data map[string]interface{}
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return &aliasDiag{
			Summary: "Error Parsing Alias Data",
			Detail:  "Could not parse alias data: " + err.Error(),
		}
	}

	// Map API response fields to the model.
	if v, ok := data["id"].(float64); ok {
		model.ID = types.Int64Value(int64(v))
	}
	if v, ok := data["domain_id"].(float64); ok {
		model.DomainID = types.Int64Value(int64(v))
	}
	if v, ok := data["local_part"].(string); ok {
		model.LocalPart = types.StringValue(v)
	} else {
		model.LocalPart = types.StringNull()
	}
	if v, ok := data["forward_to"].(string); ok {
		model.ForwardTo = types.StringValue(v)
	}
	if v, ok := data["is_catchall"].(bool); ok {
		model.IsCatchall = types.BoolValue(v)
	} else {
		model.IsCatchall = types.BoolNull()
	}
	if v, ok := data["description"].(string); ok {
		model.Description = types.StringValue(v)
	} else {
		model.Description = types.StringNull()
	}

	// Computed attributes
	if v, ok := data["is_active"].(bool); ok {
		model.IsActive = types.BoolValue(v)
	} else {
		model.IsActive = types.BoolNull()
	}
	if v, ok := data["is_wildcard"].(bool); ok {
		model.IsWildcard = types.BoolValue(v)
	} else {
		model.IsWildcard = types.BoolNull()
	}
	if v, ok := data["created_at"].(string); ok {
		model.CreatedAt = types.StringValue(v)
	} else {
		model.CreatedAt = types.StringNull()
	}
	if v, ok := data["updated_at"].(string); ok {
		model.UpdatedAt = types.StringValue(v)
	} else {
		model.UpdatedAt = types.StringNull()
	}

	return nil
}
