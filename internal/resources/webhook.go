// Copyright (c) RacterMX
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-ractermx/internal/client"
)

// Ensure WebhookResource satisfies the resource interfaces.
var (
	_ resource.Resource                = &WebhookResource{}
	_ resource.ResourceWithImportState = &WebhookResource{}
)

// WebhookResource implements the ractermx_webhook resource.
type WebhookResource struct {
	client *client.Client
}

// WebhookResourceModel maps the resource schema to a Go struct.
type WebhookResourceModel struct {
	ID             types.Int64  `tfsdk:"id"`
	URL            types.String `tfsdk:"url"`
	Events         types.List   `tfsdk:"events"`
	CustomHeaders  types.Map    `tfsdk:"custom_headers"`
	TimeoutSeconds types.Int64  `tfsdk:"timeout_seconds"`
	BatchEnabled   types.Bool   `tfsdk:"batch_enabled"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	Secret         types.String `tfsdk:"secret"`
	CreatedAt      types.String `tfsdk:"created_at"`
}

// NewWebhookResource returns a new resource.Resource for the webhook resource.
func NewWebhookResource() resource.Resource {
	return &WebhookResource{}
}

func (r *WebhookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

func (r *WebhookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a RacterMX webhook endpoint.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The numeric ID of the webhook.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"url": schema.StringAttribute{
				Description: "The URL to receive webhook events.",
				Required:    true,
			},
			"events": schema.ListAttribute{
				Description: "List of event types to subscribe to (e.g., sent, delivered, bounced, failed, unsubscribed).",
				Required:    true,
				ElementType: types.StringType,
			},
			"custom_headers": schema.MapAttribute{
				Description: "Custom headers to include in webhook requests.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"timeout_seconds": schema.Int64Attribute{
				Description: "Request timeout in seconds (5-30, default 10).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(10),
			},
			"batch_enabled": schema.BoolAttribute{
				Description: "Whether batch delivery is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the webhook is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"secret": schema.StringAttribute{
				Description: "The webhook signing secret. Only available on creation; will be empty after import.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the webhook was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *WebhookResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WebhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WebhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build request body from the plan.
	body := map[string]interface{}{
		"url": plan.URL.ValueString(),
	}

	// Convert events list to []string.
	var events []string
	resp.Diagnostics.Append(plan.Events.ElementsAs(ctx, &events, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body["events"] = events

	// Convert custom_headers map to map[string]string.
	if !plan.CustomHeaders.IsNull() && !plan.CustomHeaders.IsUnknown() {
		var headers map[string]string
		resp.Diagnostics.Append(plan.CustomHeaders.ElementsAs(ctx, &headers, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		body["custom_headers"] = headers
	}

	if !plan.TimeoutSeconds.IsNull() && !plan.TimeoutSeconds.IsUnknown() {
		body["timeout_seconds"] = plan.TimeoutSeconds.ValueInt64()
	}
	if !plan.BatchEnabled.IsNull() && !plan.BatchEnabled.IsUnknown() {
		body["batch_enabled"] = plan.BatchEnabled.ValueBool()
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		body["enabled"] = plan.Enabled.ValueBool()
	}

	result, err := r.client.Post(ctx, "/webhooks", body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Webhook",
			"Could not create webhook, unexpected error: "+err.Error(),
		)
		return
	}

	// Parse the API response envelope: {"data": {...}}
	var state WebhookResourceModel
	if diags := parseWebhookResponse(ctx, result, &state); diags != nil {
		resp.Diagnostics.AddError(diags.Summary, diags.Detail)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *WebhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state WebhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	webhookID := state.ID.ValueInt64()

	// Webhooks have no individual GET endpoint. List all and match by ID.
	result, err := r.client.Get(ctx, "/webhooks", false)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Webhook",
			"Could not read webhooks: "+err.Error(),
		)
		return
	}

	webhook, err := findWebhookByID(result, webhookID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Webhooks",
			"Could not parse webhooks response: "+err.Error(),
		)
		return
	}

	if webhook == nil {
		// Webhook not found — removed out-of-band.
		resp.State.RemoveResource(ctx)
		return
	}

	var refreshed WebhookResourceModel
	if diags := parseWebhookData(ctx, webhook, &refreshed); diags != nil {
		resp.Diagnostics.AddError(diags.Summary, diags.Detail)
		return
	}

	// Preserve the secret from state since the API does not return it on read.
	refreshed.Secret = state.Secret

	resp.Diagnostics.Append(resp.State.Set(ctx, &refreshed)...)
}

func (r *WebhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan WebhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state WebhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	webhookID := strconv.FormatInt(state.ID.ValueInt64(), 10)

	// Build request body.
	body := map[string]interface{}{
		"url": plan.URL.ValueString(),
	}

	// Convert events list to []string.
	var events []string
	resp.Diagnostics.Append(plan.Events.ElementsAs(ctx, &events, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body["events"] = events

	// Convert custom_headers map to map[string]string.
	if !plan.CustomHeaders.IsNull() && !plan.CustomHeaders.IsUnknown() {
		var headers map[string]string
		resp.Diagnostics.Append(plan.CustomHeaders.ElementsAs(ctx, &headers, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		body["custom_headers"] = headers
	}

	if !plan.TimeoutSeconds.IsNull() && !plan.TimeoutSeconds.IsUnknown() {
		body["timeout_seconds"] = plan.TimeoutSeconds.ValueInt64()
	}
	if !plan.BatchEnabled.IsNull() && !plan.BatchEnabled.IsUnknown() {
		body["batch_enabled"] = plan.BatchEnabled.ValueBool()
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		body["enabled"] = plan.Enabled.ValueBool()
	}

	result, err := r.client.Put(ctx, "/webhooks/"+webhookID, body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Webhook",
			"Could not update webhook ID "+webhookID+": "+err.Error(),
		)
		return
	}

	var updated WebhookResourceModel
	if diags := parseWebhookResponse(ctx, result, &updated); diags != nil {
		resp.Diagnostics.AddError(diags.Summary, diags.Detail)
		return
	}

	// Preserve the secret from state since the API does not return it on update.
	updated.Secret = state.Secret

	resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
}

func (r *WebhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state WebhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	webhookID := strconv.FormatInt(state.ID.ValueInt64(), 10)

	err := r.client.Delete(ctx, "/webhooks/"+webhookID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Webhook",
			"Could not delete webhook ID "+webhookID+": "+err.Error(),
		)
		return
	}
}

func (r *WebhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID as an integer.
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected a numeric webhook ID, got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

// webhookDiag is a simple struct for returning parse errors.
type webhookDiag struct {
	Summary string
	Detail  string
}

// findWebhookByID searches the webhooks list response for a webhook matching the given ID.
// Returns nil if no match is found.
func findWebhookByID(body []byte, webhookID int64) (map[string]interface{}, error) {
	var envelope struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("could not parse webhooks response: %w", err)
	}

	for _, raw := range envelope.Data {
		var webhook map[string]interface{}
		if err := json.Unmarshal(raw, &webhook); err != nil {
			continue
		}

		if id, ok := webhook["id"].(float64); ok && int64(id) == webhookID {
			return webhook, nil
		}
	}

	return nil, nil
}

// parseWebhookResponse parses the API response envelope {"data": {...}} and maps to a WebhookResourceModel.
func parseWebhookResponse(ctx context.Context, body []byte, model *WebhookResourceModel) *webhookDiag {
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return &webhookDiag{
			Summary: "Error Parsing Webhook Response",
			Detail:  "Could not parse API response envelope: " + err.Error(),
		}
	}

	var data map[string]interface{}
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return &webhookDiag{
			Summary: "Error Parsing Webhook Data",
			Detail:  "Could not parse webhook data: " + err.Error(),
		}
	}

	return parseWebhookData(ctx, data, model)
}

// parseWebhookData maps a webhook data object to a WebhookResourceModel.
func parseWebhookData(ctx context.Context, data map[string]interface{}, model *WebhookResourceModel) *webhookDiag {
	if v, ok := data["id"].(float64); ok {
		model.ID = types.Int64Value(int64(v))
	}
	if v, ok := data["url"].(string); ok {
		model.URL = types.StringValue(v)
	}

	// Parse events array.
	if eventsRaw, ok := data["events"].([]interface{}); ok {
		eventValues := make([]attr.Value, 0, len(eventsRaw))
		for _, e := range eventsRaw {
			if s, ok := e.(string); ok {
				eventValues = append(eventValues, types.StringValue(s))
			}
		}
		eventsList, diags := types.ListValue(types.StringType, eventValues)
		if diags.HasError() {
			return &webhookDiag{
				Summary: "Error Parsing Webhook Events",
				Detail:  "Could not construct events list from API response.",
			}
		}
		model.Events = eventsList
	} else {
		model.Events = types.ListNull(types.StringType)
	}

	// Parse custom_headers map.
	if headersRaw, ok := data["custom_headers"].(map[string]interface{}); ok && len(headersRaw) > 0 {
		headerValues := make(map[string]attr.Value, len(headersRaw))
		for k, v := range headersRaw {
			if s, ok := v.(string); ok {
				headerValues[k] = types.StringValue(s)
			}
		}
		headersMap, diags := types.MapValue(types.StringType, headerValues)
		if diags.HasError() {
			return &webhookDiag{
				Summary: "Error Parsing Webhook Custom Headers",
				Detail:  "Could not construct custom_headers map from API response.",
			}
		}
		model.CustomHeaders = headersMap
	} else {
		model.CustomHeaders = types.MapNull(types.StringType)
	}

	if v, ok := data["timeout_seconds"].(float64); ok {
		model.TimeoutSeconds = types.Int64Value(int64(v))
	}
	if v, ok := data["batch_enabled"].(bool); ok {
		model.BatchEnabled = types.BoolValue(v)
	}
	if v, ok := data["enabled"].(bool); ok {
		model.Enabled = types.BoolValue(v)
	}

	// Secret is only returned on create.
	if v, ok := data["secret"].(string); ok && v != "" {
		model.Secret = types.StringValue(v)
	} else {
		// If not present in response, set to null (will be preserved from state in Read).
		model.Secret = types.StringNull()
	}

	if v, ok := data["created_at"].(string); ok {
		model.CreatedAt = types.StringValue(v)
	} else {
		model.CreatedAt = types.StringNull()
	}

	return nil
}
