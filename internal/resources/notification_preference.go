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

// Ensure NotificationPreferenceResource satisfies the resource interfaces.
var (
	_ resource.Resource                = &NotificationPreferenceResource{}
	_ resource.ResourceWithImportState = &NotificationPreferenceResource{}
)

// NotificationPreferenceResource implements the ractermx_domain_notification_preference resource.
type NotificationPreferenceResource struct {
	client *client.Client
}

// NotificationPreferenceResourceModel maps the resource schema to a Go struct.
type NotificationPreferenceResourceModel struct {
	DomainID    types.Int64  `tfsdk:"domain_id"`
	Muted       types.Bool   `tfsdk:"muted"`
	MinPriority types.String `tfsdk:"min_priority"`
}

// NewNotificationPreferenceResource returns a new resource.Resource for the notification preference resource.
func NewNotificationPreferenceResource() resource.Resource {
	return &NotificationPreferenceResource{}
}

func (r *NotificationPreferenceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_notification_preference"
}

func (r *NotificationPreferenceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages per-domain notification preferences in RacterMX.",
		Attributes: map[string]schema.Attribute{
			"domain_id": schema.Int64Attribute{
				Description: "The numeric ID of the domain. Changing this forces a new resource.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"muted": schema.BoolAttribute{
				Description: "Whether notifications are muted for this domain.",
				Optional:    true,
				Computed:    true,
			},
			"min_priority": schema.StringAttribute{
				Description: "The minimum priority threshold for notifications.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func (r *NotificationPreferenceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NotificationPreferenceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NotificationPreferenceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := plan.DomainID.ValueInt64()
	body := buildNotificationPreferenceBody(&plan)

	result, err := r.client.Post(ctx, fmt.Sprintf("/domains/%d/notification-preferences", domainID), body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Notification Preference",
			fmt.Sprintf("Could not create notification preference for domain %d: %s", domainID, err.Error()),
		)
		return
	}

	var state NotificationPreferenceResourceModel
	state.DomainID = plan.DomainID
	if d := parseNotificationPreferenceResponse(result, &state); d != nil {
		resp.Diagnostics.AddError(d.Summary, d.Detail)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *NotificationPreferenceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NotificationPreferenceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := state.DomainID.ValueInt64()

	result, err := r.client.Get(ctx, fmt.Sprintf("/domains/%d/notification-preferences", domainID), true)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Notification Preference",
			fmt.Sprintf("Could not read notification preference for domain %d: %s", domainID, err.Error()),
		)
		return
	}

	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	var refreshed NotificationPreferenceResourceModel
	refreshed.DomainID = state.DomainID
	if d := parseNotificationPreferenceResponse(result, &refreshed); d != nil {
		resp.Diagnostics.AddError(d.Summary, d.Detail)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &refreshed)...)
}

func (r *NotificationPreferenceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan NotificationPreferenceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := plan.DomainID.ValueInt64()
	body := buildNotificationPreferenceBody(&plan)

	// Upsert: Create and Update use the same endpoint.
	result, err := r.client.Post(ctx, fmt.Sprintf("/domains/%d/notification-preferences", domainID), body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Notification Preference",
			fmt.Sprintf("Could not update notification preference for domain %d: %s", domainID, err.Error()),
		)
		return
	}

	var updated NotificationPreferenceResourceModel
	updated.DomainID = plan.DomainID
	if d := parseNotificationPreferenceResponse(result, &updated); d != nil {
		resp.Diagnostics.AddError(d.Summary, d.Detail)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
}

func (r *NotificationPreferenceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NotificationPreferenceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := state.DomainID.ValueInt64()

	// Reset to defaults rather than deleting.
	body := map[string]interface{}{
		"muted":        false,
		"min_priority": nil,
	}

	_, err := r.client.Post(ctx, fmt.Sprintf("/domains/%d/notification-preferences", domainID), body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Resetting Notification Preference",
			fmt.Sprintf("Could not reset notification preference for domain %d: %s", domainID, err.Error()),
		)
		return
	}
}

func (r *NotificationPreferenceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected a numeric domain ID, got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), id)...)
}

// notificationPreferenceDiag is a simple struct for returning parse errors.
type notificationPreferenceDiag struct {
	Summary string
	Detail  string
}

// buildNotificationPreferenceBody constructs the API request body from the plan model.
func buildNotificationPreferenceBody(plan *NotificationPreferenceResourceModel) map[string]interface{} {
	body := map[string]interface{}{}

	if !plan.Muted.IsNull() && !plan.Muted.IsUnknown() {
		body["muted"] = plan.Muted.ValueBool()
	}
	if !plan.MinPriority.IsNull() && !plan.MinPriority.IsUnknown() {
		body["min_priority"] = plan.MinPriority.ValueString()
	} else {
		body["min_priority"] = nil
	}

	return body
}

// parseNotificationPreferenceResponse parses the API response and maps to a NotificationPreferenceResourceModel.
func parseNotificationPreferenceResponse(body []byte, model *NotificationPreferenceResourceModel) *notificationPreferenceDiag {
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return &notificationPreferenceDiag{
			Summary: "Error Parsing Notification Preference Response",
			Detail:  "Could not parse API response envelope: " + err.Error(),
		}
	}

	var data map[string]interface{}
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return &notificationPreferenceDiag{
			Summary: "Error Parsing Notification Preference Data",
			Detail:  "Could not parse notification preference data: " + err.Error(),
		}
	}

	if v, ok := data["muted"].(bool); ok {
		model.Muted = types.BoolValue(v)
	} else {
		model.Muted = types.BoolValue(false)
	}
	if v, ok := data["min_priority"].(string); ok {
		model.MinPriority = types.StringValue(v)
	} else {
		model.MinPriority = types.StringNull()
	}

	return nil
}
