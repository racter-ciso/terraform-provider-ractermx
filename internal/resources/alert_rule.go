// Copyright (c) RacterMX
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-ractermx/internal/client"
)

// Ensure AlertRuleResource satisfies the resource interfaces.
var (
	_ resource.Resource                = &AlertRuleResource{}
	_ resource.ResourceWithImportState = &AlertRuleResource{}
)

// AlertRuleResource implements the ractermx_alert_rule resource.
type AlertRuleResource struct {
	client *client.Client
}

// AlertRuleChannelModel represents a notification channel in an alert rule.
type AlertRuleChannelModel struct {
	ChannelType       types.String `tfsdk:"channel_type"`
	WebhookEndpointID types.Int64  `tfsdk:"webhook_endpoint_id"`
	EmailAddress      types.String `tfsdk:"email_address"`
}

// AlertRuleResourceModel maps the resource schema to a Go struct.
type AlertRuleResourceModel struct {
	ID              types.Int64  `tfsdk:"id"`
	DomainID        types.Int64  `tfsdk:"domain_id"`
	Name            types.String `tfsdk:"name"`
	AlertType       types.String `tfsdk:"alert_type"`
	Condition       types.String `tfsdk:"condition"`
	ThresholdValue  types.String `tfsdk:"threshold_value"`
	CooldownMinutes types.Int64  `tfsdk:"cooldown_minutes"`
	Enabled         types.Bool   `tfsdk:"enabled"`
	Channels        types.List   `tfsdk:"channels"`
	CreatedAt       types.String `tfsdk:"created_at"`
}

// NewAlertRuleResource returns a new resource.Resource for the alert rule resource.
func NewAlertRuleResource() resource.Resource {
	return &AlertRuleResource{}
}

func (r *AlertRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_rule"
}

func (r *AlertRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a RacterMX alert rule.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The numeric ID of the alert rule.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"domain_id": schema.Int64Attribute{
				Description: "The numeric ID of the domain this alert rule applies to.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the alert rule (1-100 chars).",
				Required:    true,
			},
			"alert_type": schema.StringAttribute{
				Description: "The type of alert (deliverability_score, blacklist_change, security_posture, dmarc_compliance).",
				Required:    true,
			},
			"condition": schema.StringAttribute{
				Description: "The condition for the alert (below, above, equals, any_change).",
				Required:    true,
			},
			"threshold_value": schema.StringAttribute{
				Description: "The threshold value for the alert (max 50 chars). Required for some alert types.",
				Optional:    true,
			},
			"cooldown_minutes": schema.Int64Attribute{
				Description: "Cooldown period in minutes between alerts (15-1440, default 60).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(60),
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the alert rule is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"channels": schema.ListNestedAttribute{
				Description: "Notification channels for this alert rule (1-3 items).",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"channel_type": schema.StringAttribute{
							Description: "The type of channel (webhook or email).",
							Required:    true,
						},
						"webhook_endpoint_id": schema.Int64Attribute{
							Description: "The webhook endpoint ID (required when channel_type is webhook).",
							Optional:    true,
						},
						"email_address": schema.StringAttribute{
							Description: "The email address (required when channel_type is email).",
							Optional:    true,
						},
					},
				},
				Validators: []validator.List{
					listvalidator.SizeBetween(1, 3),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the alert rule was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *AlertRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AlertRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AlertRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Cross-field validation.
	var thresholdPtr *string
	if !plan.ThresholdValue.IsNull() && !plan.ThresholdValue.IsUnknown() {
		v := plan.ThresholdValue.ValueString()
		thresholdPtr = &v
	}
	if err := ValidateAlertRule(plan.AlertType.ValueString(), plan.Condition.ValueString(), thresholdPtr); err != nil {
		resp.Diagnostics.AddError("Alert Rule Validation Error", err.Error())
		return
	}

	body, diags := buildAlertRuleRequestBody(ctx, &plan)
	if diags != nil {
		resp.Diagnostics.AddError(diags.Summary, diags.Detail)
		return
	}

	result, err := r.client.Post(ctx, "/alert-rules", body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Alert Rule",
			"Could not create alert rule, unexpected error: "+err.Error(),
		)
		return
	}

	var state AlertRuleResourceModel
	if d := parseAlertRuleResponse(ctx, result, &state); d != nil {
		resp.Diagnostics.AddError(d.Summary, d.Detail)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AlertRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AlertRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ruleID := strconv.FormatInt(state.ID.ValueInt64(), 10)
	result, err := r.client.Get(ctx, "/alert-rules/"+ruleID, true)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Alert Rule",
			"Could not read alert rule ID "+ruleID+": "+err.Error(),
		)
		return
	}

	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	var refreshed AlertRuleResourceModel
	if d := parseAlertRuleResponse(ctx, result, &refreshed); d != nil {
		resp.Diagnostics.AddError(d.Summary, d.Detail)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &refreshed)...)
}

func (r *AlertRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AlertRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state AlertRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Cross-field validation.
	var thresholdPtr *string
	if !plan.ThresholdValue.IsNull() && !plan.ThresholdValue.IsUnknown() {
		v := plan.ThresholdValue.ValueString()
		thresholdPtr = &v
	}
	if err := ValidateAlertRule(plan.AlertType.ValueString(), plan.Condition.ValueString(), thresholdPtr); err != nil {
		resp.Diagnostics.AddError("Alert Rule Validation Error", err.Error())
		return
	}

	ruleID := strconv.FormatInt(state.ID.ValueInt64(), 10)

	body, diags := buildAlertRuleRequestBody(ctx, &plan)
	if diags != nil {
		resp.Diagnostics.AddError(diags.Summary, diags.Detail)
		return
	}

	result, err := r.client.Patch(ctx, "/alert-rules/"+ruleID, body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Alert Rule",
			"Could not update alert rule ID "+ruleID+": "+err.Error(),
		)
		return
	}

	var updated AlertRuleResourceModel
	if d := parseAlertRuleResponse(ctx, result, &updated); d != nil {
		resp.Diagnostics.AddError(d.Summary, d.Detail)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
}

func (r *AlertRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AlertRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ruleID := strconv.FormatInt(state.ID.ValueInt64(), 10)

	err := r.client.Delete(ctx, "/alert-rules/"+ruleID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Alert Rule",
			"Could not delete alert rule ID "+ruleID+": "+err.Error(),
		)
		return
	}
}

func (r *AlertRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected a numeric alert rule ID, got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

// alertRuleDiag is a simple struct for returning parse errors.
type alertRuleDiag struct {
	Summary string
	Detail  string
}

// buildAlertRuleRequestBody constructs the API request body from the plan model.
func buildAlertRuleRequestBody(ctx context.Context, plan *AlertRuleResourceModel) (map[string]interface{}, *alertRuleDiag) {
	body := map[string]interface{}{
		"domain_id":  plan.DomainID.ValueInt64(),
		"name":       plan.Name.ValueString(),
		"alert_type": plan.AlertType.ValueString(),
		"condition":  plan.Condition.ValueString(),
	}

	if !plan.ThresholdValue.IsNull() && !plan.ThresholdValue.IsUnknown() {
		body["threshold_value"] = plan.ThresholdValue.ValueString()
	}
	if !plan.CooldownMinutes.IsNull() && !plan.CooldownMinutes.IsUnknown() {
		body["cooldown_minutes"] = plan.CooldownMinutes.ValueInt64()
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		body["enabled"] = plan.Enabled.ValueBool()
	}

	// Convert channels list to API format.
	var channels []AlertRuleChannelModel
	diags := plan.Channels.ElementsAs(ctx, &channels, false)
	if diags.HasError() {
		return nil, &alertRuleDiag{
			Summary: "Error Reading Channels",
			Detail:  "Could not read channels from plan.",
		}
	}

	channelsList := make([]map[string]interface{}, 0, len(channels))
	for _, ch := range channels {
		chMap := map[string]interface{}{
			"channel_type": ch.ChannelType.ValueString(),
		}
		if !ch.WebhookEndpointID.IsNull() && !ch.WebhookEndpointID.IsUnknown() {
			chMap["webhook_endpoint_id"] = ch.WebhookEndpointID.ValueInt64()
		}
		if !ch.EmailAddress.IsNull() && !ch.EmailAddress.IsUnknown() {
			chMap["email_address"] = ch.EmailAddress.ValueString()
		}
		channelsList = append(channelsList, chMap)
	}
	body["channels"] = channelsList

	return body, nil
}

// parseAlertRuleResponse parses the API response envelope {"data": {...}} and maps to an AlertRuleResourceModel.
func parseAlertRuleResponse(ctx context.Context, body []byte, model *AlertRuleResourceModel) *alertRuleDiag {
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return &alertRuleDiag{
			Summary: "Error Parsing Alert Rule Response",
			Detail:  "Could not parse API response envelope: " + err.Error(),
		}
	}

	var data map[string]interface{}
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return &alertRuleDiag{
			Summary: "Error Parsing Alert Rule Data",
			Detail:  "Could not parse alert rule data: " + err.Error(),
		}
	}

	if v, ok := data["id"].(float64); ok {
		model.ID = types.Int64Value(int64(v))
	}
	if v, ok := data["domain_id"].(float64); ok {
		model.DomainID = types.Int64Value(int64(v))
	}
	if v, ok := data["name"].(string); ok {
		model.Name = types.StringValue(v)
	}
	if v, ok := data["alert_type"].(string); ok {
		model.AlertType = types.StringValue(v)
	}
	if v, ok := data["condition"].(string); ok {
		model.Condition = types.StringValue(v)
	}
	if v, ok := data["threshold_value"].(string); ok {
		model.ThresholdValue = types.StringValue(v)
	} else {
		model.ThresholdValue = types.StringNull()
	}
	if v, ok := data["cooldown_minutes"].(float64); ok {
		model.CooldownMinutes = types.Int64Value(int64(v))
	}
	if v, ok := data["enabled"].(bool); ok {
		model.Enabled = types.BoolValue(v)
	}
	if v, ok := data["created_at"].(string); ok {
		model.CreatedAt = types.StringValue(v)
	} else {
		model.CreatedAt = types.StringNull()
	}

	// Parse channels array.
	if channelsRaw, ok := data["channels"].([]interface{}); ok {
		channelModels := make([]AlertRuleChannelModel, 0, len(channelsRaw))
		for _, chRaw := range channelsRaw {
			ch, ok := chRaw.(map[string]interface{})
			if !ok {
				continue
			}
			chModel := AlertRuleChannelModel{}
			if v, ok := ch["channel_type"].(string); ok {
				chModel.ChannelType = types.StringValue(v)
			}
			if v, ok := ch["webhook_endpoint_id"].(float64); ok {
				chModel.WebhookEndpointID = types.Int64Value(int64(v))
			} else {
				chModel.WebhookEndpointID = types.Int64Null()
			}
			if v, ok := ch["email_address"].(string); ok {
				chModel.EmailAddress = types.StringValue(v)
			} else {
				chModel.EmailAddress = types.StringNull()
			}
			channelModels = append(channelModels, chModel)
		}

		channelObjType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"channel_type":        types.StringType,
				"webhook_endpoint_id": types.Int64Type,
				"email_address":       types.StringType,
			},
		}

		channelObjValues := make([]attr.Value, 0, len(channelModels))
		for _, ch := range channelModels {
			objVal, diags := types.ObjectValue(
				channelObjType.AttrTypes,
				map[string]attr.Value{
					"channel_type":        ch.ChannelType,
					"webhook_endpoint_id": ch.WebhookEndpointID,
					"email_address":       ch.EmailAddress,
				},
			)
			if diags.HasError() {
				return &alertRuleDiag{
					Summary: "Error Constructing Channel Object",
					Detail:  "Could not construct channel object from API response.",
				}
			}
			channelObjValues = append(channelObjValues, objVal)
		}

		channelsList, diags := types.ListValue(channelObjType, channelObjValues)
		if diags.HasError() {
			return &alertRuleDiag{
				Summary: "Error Constructing Channels List",
				Detail:  "Could not construct channels list from API response.",
			}
		}
		model.Channels = channelsList
	} else {
		channelObjType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"channel_type":        types.StringType,
				"webhook_endpoint_id": types.Int64Type,
				"email_address":       types.StringType,
			},
		}
		model.Channels = types.ListNull(channelObjType)
	}

	return nil
}
