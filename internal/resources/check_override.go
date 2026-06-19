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

// Ensure CheckOverrideResource satisfies the resource interfaces.
var (
	_ resource.Resource                = &CheckOverrideResource{}
	_ resource.ResourceWithImportState = &CheckOverrideResource{}
)

// CheckOverrideResource implements the ractermx_check_override resource.
type CheckOverrideResource struct {
	client *client.Client
}

// CheckOverrideResourceModel maps the resource schema to a Go struct.
type CheckOverrideResourceModel struct {
	ID               types.String `tfsdk:"id"`
	DomainID         types.Int64  `tfsdk:"domain_id"`
	CheckID          types.String `tfsdk:"check_id"`
	Enabled          types.Bool   `tfsdk:"enabled"`
	SeverityOverride types.String `tfsdk:"severity_override"`
}

// NewCheckOverrideResource returns a new resource.Resource for the check override resource.
func NewCheckOverrideResource() resource.Resource {
	return &CheckOverrideResource{}
}

func (r *CheckOverrideResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_check_override"
}

func (r *CheckOverrideResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a per-domain security check override in RacterMX.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The composite ID of the check override ({domain_id}/{check_id}).",
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
			"check_id": schema.StringAttribute{
				Description: "The ID of the security check to override. Changing this forces a new resource.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the check is enabled for this domain.",
				Optional:    true,
				Computed:    true,
			},
			"severity_override": schema.StringAttribute{
				Description: "Override severity level (critical, high, medium, low, informational).",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func (r *CheckOverrideResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CheckOverrideResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CheckOverrideResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := plan.DomainID.ValueInt64()
	checkID := plan.CheckID.ValueString()
	body := buildCheckOverrideBody(&plan)

	result, err := r.client.Put(ctx, fmt.Sprintf("/domains/%d/check-overrides/%s", domainID, checkID), body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Check Override",
			fmt.Sprintf("Could not create check override for domain %d, check %s: %s", domainID, checkID, err.Error()),
		)
		return
	}

	var state CheckOverrideResourceModel
	state.DomainID = plan.DomainID
	state.CheckID = plan.CheckID
	state.ID = types.StringValue(FormatCheckOverrideID(domainID, checkID))
	if d := parseCheckOverrideResponse(result, &state); d != nil {
		resp.Diagnostics.AddError(d.Summary, d.Detail)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *CheckOverrideResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CheckOverrideResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := state.DomainID.ValueInt64()
	checkID := state.CheckID.ValueString()

	// Read from the check catalog and find the override for this domain+check.
	result, err := r.client.Get(ctx, fmt.Sprintf("/domains/%d/check-overrides/%s", domainID, checkID), true)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Check Override",
			fmt.Sprintf("Could not read check override for domain %d, check %s: %s", domainID, checkID, err.Error()),
		)
		return
	}

	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	var refreshed CheckOverrideResourceModel
	refreshed.DomainID = state.DomainID
	refreshed.CheckID = state.CheckID
	refreshed.ID = types.StringValue(FormatCheckOverrideID(domainID, checkID))
	if d := parseCheckOverrideResponse(result, &refreshed); d != nil {
		resp.Diagnostics.AddError(d.Summary, d.Detail)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &refreshed)...)
}

func (r *CheckOverrideResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan CheckOverrideResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := plan.DomainID.ValueInt64()
	checkID := plan.CheckID.ValueString()
	body := buildCheckOverrideBody(&plan)

	// Upsert: Create and Update use the same endpoint.
	result, err := r.client.Put(ctx, fmt.Sprintf("/domains/%d/check-overrides/%s", domainID, checkID), body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Check Override",
			fmt.Sprintf("Could not update check override for domain %d, check %s: %s", domainID, checkID, err.Error()),
		)
		return
	}

	var updated CheckOverrideResourceModel
	updated.DomainID = plan.DomainID
	updated.CheckID = plan.CheckID
	updated.ID = types.StringValue(FormatCheckOverrideID(domainID, checkID))
	if d := parseCheckOverrideResponse(result, &updated); d != nil {
		resp.Diagnostics.AddError(d.Summary, d.Detail)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
}

func (r *CheckOverrideResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CheckOverrideResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := state.DomainID.ValueInt64()
	checkID := state.CheckID.ValueString()

	// Reset to defaults by sending null values.
	body := map[string]interface{}{
		"enabled":           nil,
		"severity_override": nil,
	}

	_, err := r.client.Put(ctx, fmt.Sprintf("/domains/%d/check-overrides/%s", domainID, checkID), body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Resetting Check Override",
			fmt.Sprintf("Could not reset check override for domain %d, check %s: %s", domainID, checkID, err.Error()),
		)
		return
	}
}

func (r *CheckOverrideResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	domainID, checkID, err := ParseCheckOverrideID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Could not parse check override import ID: %s", err.Error()),
		)
		return
	}

	state := CheckOverrideResourceModel{
		ID:       types.StringValue(FormatCheckOverrideID(domainID, checkID)),
		DomainID: types.Int64Value(domainID),
		CheckID:  types.StringValue(checkID),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// checkOverrideDiag is a simple struct for returning parse errors.
type checkOverrideDiag struct {
	Summary string
	Detail  string
}

// buildCheckOverrideBody constructs the API request body from the plan model.
func buildCheckOverrideBody(plan *CheckOverrideResourceModel) map[string]interface{} {
	body := map[string]interface{}{}

	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		body["enabled"] = plan.Enabled.ValueBool()
	}
	if !plan.SeverityOverride.IsNull() && !plan.SeverityOverride.IsUnknown() {
		body["severity_override"] = plan.SeverityOverride.ValueString()
	}

	return body
}

// parseCheckOverrideResponse parses the API response and maps to a CheckOverrideResourceModel.
func parseCheckOverrideResponse(body []byte, model *CheckOverrideResourceModel) *checkOverrideDiag {
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return &checkOverrideDiag{
			Summary: "Error Parsing Check Override Response",
			Detail:  "Could not parse API response envelope: " + err.Error(),
		}
	}

	var data map[string]interface{}
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return &checkOverrideDiag{
			Summary: "Error Parsing Check Override Data",
			Detail:  "Could not parse check override data: " + err.Error(),
		}
	}

	// Parse domain_id if present (may come from API).
	if v, ok := data["domain_id"].(float64); ok {
		model.DomainID = types.Int64Value(int64(v))
		// Reconstruct composite ID if check_id is also available.
		if cid, ok := data["check_id"].(string); ok {
			model.CheckID = types.StringValue(cid)
			model.ID = types.StringValue(FormatCheckOverrideID(int64(v), cid))
		}
	}

	if v, ok := data["enabled"].(bool); ok {
		model.Enabled = types.BoolValue(v)
	} else {
		model.Enabled = types.BoolNull()
	}
	if v, ok := data["severity_override"].(string); ok {
		model.SeverityOverride = types.StringValue(v)
	} else {
		model.SeverityOverride = types.StringNull()
	}

	return nil
}
