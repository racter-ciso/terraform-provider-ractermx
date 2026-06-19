// Copyright (c) RacterMX
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-ractermx/internal/client"
)

// Ensure RetentionPolicyResource satisfies the resource interfaces.
var (
	_ resource.Resource                = &RetentionPolicyResource{}
	_ resource.ResourceWithImportState = &RetentionPolicyResource{}
)

// RetentionPolicyResource implements the ractermx_retention_policy resource.
type RetentionPolicyResource struct {
	client *client.Client
}

// RetentionPolicyResourceModel maps the resource schema to a Go struct.
type RetentionPolicyResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	MetadataRetentionDays   types.Int64  `tfsdk:"metadata_retention_days"`
	EventSpecificRetention  types.Map    `tfsdk:"event_specific_retention"`
	UpdatedAt               types.String `tfsdk:"updated_at"`
}

// NewRetentionPolicyResource returns a new resource.Resource for the retention policy resource.
func NewRetentionPolicyResource() resource.Resource {
	return &RetentionPolicyResource{}
}

func (r *RetentionPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_retention_policy"
}

func (r *RetentionPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the RacterMX email log retention policy. This is a singleton resource (one per organization).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The resource identifier. Always 'default' for the singleton retention policy.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"metadata_retention_days": schema.Int64Attribute{
				Description: "Number of days to retain email metadata (7-2555).",
				Required:    true,
			},
			"event_specific_retention": schema.MapAttribute{
				Description: "Per-event retention overrides. Map of event name to retention days (7-2555).",
				Optional:    true,
				ElementType: types.Int64Type,
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the retention policy was last updated.",
				Computed:    true,
			},
		},
	}
}

func (r *RetentionPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RetentionPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RetentionPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"metadata_retention_days": plan.MetadataRetentionDays.ValueInt64(),
	}

	if !plan.EventSpecificRetention.IsNull() && !plan.EventSpecificRetention.IsUnknown() {
		var retention map[string]int64
		resp.Diagnostics.Append(plan.EventSpecificRetention.ElementsAs(ctx, &retention, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		body["event_specific_retention"] = retention
	}

	result, err := r.client.Put(ctx, "/retention-policy", body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Retention Policy",
			"Could not create retention policy, unexpected error: "+err.Error(),
		)
		return
	}

	var state RetentionPolicyResourceModel
	if diags := parseRetentionPolicyResponse(result, &state); diags != nil {
		resp.Diagnostics.AddError(diags.Summary, diags.Detail)
		return
	}

	// The ID is always "default" for the singleton resource.
	state.ID = types.StringValue("default")

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RetentionPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RetentionPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Get(ctx, "/retention-policy", true)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Retention Policy",
			"Could not read retention policy: "+err.Error(),
		)
		return
	}

	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	var refreshed RetentionPolicyResourceModel
	if diags := parseRetentionPolicyResponse(result, &refreshed); diags != nil {
		resp.Diagnostics.AddError(diags.Summary, diags.Detail)
		return
	}

	refreshed.ID = types.StringValue("default")

	resp.Diagnostics.Append(resp.State.Set(ctx, &refreshed)...)
}

func (r *RetentionPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RetentionPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"metadata_retention_days": plan.MetadataRetentionDays.ValueInt64(),
	}

	if !plan.EventSpecificRetention.IsNull() && !plan.EventSpecificRetention.IsUnknown() {
		var retention map[string]int64
		resp.Diagnostics.Append(plan.EventSpecificRetention.ElementsAs(ctx, &retention, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		body["event_specific_retention"] = retention
	}

	result, err := r.client.Put(ctx, "/retention-policy", body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Retention Policy",
			"Could not update retention policy: "+err.Error(),
		)
		return
	}

	var updated RetentionPolicyResourceModel
	if diags := parseRetentionPolicyResponse(result, &updated); diags != nil {
		resp.Diagnostics.AddError(diags.Summary, diags.Detail)
		return
	}

	updated.ID = types.StringValue("default")

	resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
}

func (r *RetentionPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// No-op: the retention policy cannot be deleted. Log a warning.
	tflog.Warn(ctx, "Retention policy cannot be deleted from the server. "+
		"Removing from Terraform state only. The policy will remain unchanged on the server.")
}

func (r *RetentionPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != "default" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("The retention policy import ID must be \"default\", got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), "default")...)
}

// retentionPolicyDiag is a simple struct for returning parse errors.
type retentionPolicyDiag struct {
	Summary string
	Detail  string
}

// parseRetentionPolicyResponse parses the API response envelope {"data": {...}} and maps to a RetentionPolicyResourceModel.
func parseRetentionPolicyResponse(body []byte, model *RetentionPolicyResourceModel) *retentionPolicyDiag {
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return &retentionPolicyDiag{
			Summary: "Error Parsing Retention Policy Response",
			Detail:  "Could not parse API response envelope: " + err.Error(),
		}
	}

	var data map[string]interface{}
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return &retentionPolicyDiag{
			Summary: "Error Parsing Retention Policy Data",
			Detail:  "Could not parse retention policy data: " + err.Error(),
		}
	}

	if v, ok := data["metadata_retention_days"].(float64); ok {
		model.MetadataRetentionDays = types.Int64Value(int64(v))
	}

	// Parse event_specific_retention map.
	if retentionRaw, ok := data["event_specific_retention"].(map[string]interface{}); ok && len(retentionRaw) > 0 {
		retentionValues := make(map[string]attr.Value, len(retentionRaw))
		for k, v := range retentionRaw {
			if num, ok := v.(float64); ok {
				retentionValues[k] = types.Int64Value(int64(num))
			}
		}
		retentionMap, diags := types.MapValue(types.Int64Type, retentionValues)
		if diags.HasError() {
			return &retentionPolicyDiag{
				Summary: "Error Parsing Retention Policy Event Retention",
				Detail:  "Could not construct event_specific_retention map from API response.",
			}
		}
		model.EventSpecificRetention = retentionMap
	} else {
		model.EventSpecificRetention = types.MapNull(types.Int64Type)
	}

	if v, ok := data["updated_at"].(string); ok {
		model.UpdatedAt = types.StringValue(v)
	} else {
		model.UpdatedAt = types.StringNull()
	}

	return nil
}
