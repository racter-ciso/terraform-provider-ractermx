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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-ractermx/internal/client"
)

// Ensure ApiKeyResource satisfies the resource interfaces.
var (
	_ resource.Resource                = &ApiKeyResource{}
	_ resource.ResourceWithImportState = &ApiKeyResource{}
)

// ApiKeyResource implements the ractermx_api_key resource.
type ApiKeyResource struct {
	client *client.Client
}

// ApiKeyResourceModel maps the resource schema to a Go struct.
type ApiKeyResourceModel struct {
	ID         types.Int64  `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Scopes     types.List   `tfsdk:"scopes"`
	ExpiresAt  types.String `tfsdk:"expires_at"`
	AllowedIPs types.List   `tfsdk:"allowed_ips"`
	// Computed
	ApiKey     types.String `tfsdk:"api_key"`
	LastUsedAt types.String `tfsdk:"last_used_at"`
	CreatedAt  types.String `tfsdk:"created_at"`
}

// NewApiKeyResource returns a new resource.Resource for the API key resource.
func NewApiKeyResource() resource.Resource {
	return &ApiKeyResource{}
}

func (r *ApiKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (r *ApiKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a RacterMX API key.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The numeric ID of the API key.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "A descriptive name for the API key. Changing this forces a new resource.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"scopes": schema.ListAttribute{
				Description: "List of permission scopes for the API key. Changing this forces a new resource.",
				Required:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"expires_at": schema.StringAttribute{
				Description: "Expiration date in ISO 8601 format. Changing this forces a new resource.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"allowed_ips": schema.ListAttribute{
				Description: "List of allowed IP addresses or CIDR blocks. Changing this forces a new resource.",
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			// Computed attributes
			"api_key": schema.StringAttribute{
				Description: "The API key value. Only available on creation; will be empty after import.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_used_at": schema.StringAttribute{
				Description: "The timestamp when the API key was last used.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the API key was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ApiKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ApiKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ApiKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"name": plan.Name.ValueString(),
	}

	// Convert scopes list to []string.
	var scopes []string
	resp.Diagnostics.Append(plan.Scopes.ElementsAs(ctx, &scopes, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body["scopes"] = scopes

	if !plan.ExpiresAt.IsNull() && !plan.ExpiresAt.IsUnknown() {
		body["expires_at"] = plan.ExpiresAt.ValueString()
	}

	if !plan.AllowedIPs.IsNull() && !plan.AllowedIPs.IsUnknown() {
		var allowedIPs []string
		resp.Diagnostics.Append(plan.AllowedIPs.ElementsAs(ctx, &allowedIPs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		body["allowed_ips"] = allowedIPs
	}

	result, err := r.client.Post(ctx, "/api-keys", body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating API Key",
			"Could not create API key, unexpected error: "+err.Error(),
		)
		return
	}

	var state ApiKeyResourceModel
	if diags := parseApiKeyResponse(ctx, result, &state); diags != nil {
		resp.Diagnostics.AddError(diags.Summary, diags.Detail)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ApiKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ApiKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	keyID := state.ID.ValueInt64()

	// API keys have no individual GET endpoint. List all and match by ID.
	result, err := r.client.Get(ctx, "/api-keys", false)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading API Key",
			"Could not read API keys: "+err.Error(),
		)
		return
	}

	apiKey, err := findApiKeyByID(result, keyID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing API Keys",
			"Could not parse API keys response: "+err.Error(),
		)
		return
	}

	if apiKey == nil {
		// Key not found — removed out-of-band.
		resp.State.RemoveResource(ctx)
		return
	}

	var refreshed ApiKeyResourceModel
	if diags := parseApiKeyData(ctx, apiKey, &refreshed); diags != nil {
		resp.Diagnostics.AddError(diags.Summary, diags.Detail)
		return
	}

	// Preserve the api_key from state since the API does not return it on read.
	refreshed.ApiKey = state.ApiKey

	resp.Diagnostics.Append(resp.State.Set(ctx, &refreshed)...)
}

func (r *ApiKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// No update endpoint — all attribute changes force replacement via RequiresReplace().
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"API keys cannot be updated. Attribute changes force replacement.",
	)
}

func (r *ApiKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ApiKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	keyID := strconv.FormatInt(state.ID.ValueInt64(), 10)

	err := r.client.Delete(ctx, "/api-keys/"+keyID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting API Key",
			"Could not delete API key ID "+keyID+": "+err.Error(),
		)
		return
	}
}

func (r *ApiKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected a numeric API key ID, got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

// apiKeyDiag is a simple struct for returning parse errors.
type apiKeyDiag struct {
	Summary string
	Detail  string
}

// findApiKeyByID searches the API keys list response for a key matching the given ID.
func findApiKeyByID(body []byte, keyID int64) (map[string]interface{}, error) {
	var envelope struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("could not parse API keys response: %w", err)
	}

	for _, raw := range envelope.Data {
		var key map[string]interface{}
		if err := json.Unmarshal(raw, &key); err != nil {
			continue
		}

		if id, ok := key["id"].(float64); ok && int64(id) == keyID {
			return key, nil
		}
	}

	return nil, nil
}

// parseApiKeyResponse parses the API response envelope {"data": {...}} and maps to an ApiKeyResourceModel.
func parseApiKeyResponse(ctx context.Context, body []byte, model *ApiKeyResourceModel) *apiKeyDiag {
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return &apiKeyDiag{
			Summary: "Error Parsing API Key Response",
			Detail:  "Could not parse API response envelope: " + err.Error(),
		}
	}

	var data map[string]interface{}
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return &apiKeyDiag{
			Summary: "Error Parsing API Key Data",
			Detail:  "Could not parse API key data: " + err.Error(),
		}
	}

	return parseApiKeyData(ctx, data, model)
}

// parseApiKeyData maps an API key data object to an ApiKeyResourceModel.
func parseApiKeyData(ctx context.Context, data map[string]interface{}, model *ApiKeyResourceModel) *apiKeyDiag {
	if v, ok := data["id"].(float64); ok {
		model.ID = types.Int64Value(int64(v))
	}
	if v, ok := data["name"].(string); ok {
		model.Name = types.StringValue(v)
	}

	// Parse scopes array.
	if scopesRaw, ok := data["scopes"].([]interface{}); ok {
		scopeValues := make([]attr.Value, 0, len(scopesRaw))
		for _, s := range scopesRaw {
			if str, ok := s.(string); ok {
				scopeValues = append(scopeValues, types.StringValue(str))
			}
		}
		scopesList, diags := types.ListValue(types.StringType, scopeValues)
		if diags.HasError() {
			return &apiKeyDiag{
				Summary: "Error Parsing API Key Scopes",
				Detail:  "Could not construct scopes list from API response.",
			}
		}
		model.Scopes = scopesList
	} else {
		model.Scopes = types.ListNull(types.StringType)
	}

	if v, ok := data["expires_at"].(string); ok {
		model.ExpiresAt = types.StringValue(v)
	} else {
		model.ExpiresAt = types.StringNull()
	}

	// Parse allowed_ips array.
	if ipsRaw, ok := data["allowed_ips"].([]interface{}); ok && len(ipsRaw) > 0 {
		ipValues := make([]attr.Value, 0, len(ipsRaw))
		for _, ip := range ipsRaw {
			if str, ok := ip.(string); ok {
				ipValues = append(ipValues, types.StringValue(str))
			}
		}
		ipsList, diags := types.ListValue(types.StringType, ipValues)
		if diags.HasError() {
			return &apiKeyDiag{
				Summary: "Error Parsing API Key Allowed IPs",
				Detail:  "Could not construct allowed_ips list from API response.",
			}
		}
		model.AllowedIPs = ipsList
	} else {
		model.AllowedIPs = types.ListNull(types.StringType)
	}

	// api_key is only returned on create.
	if v, ok := data["api_key"].(string); ok && v != "" {
		model.ApiKey = types.StringValue(v)
	} else {
		model.ApiKey = types.StringNull()
	}

	if v, ok := data["last_used_at"].(string); ok {
		model.LastUsedAt = types.StringValue(v)
	} else {
		model.LastUsedAt = types.StringNull()
	}
	if v, ok := data["created_at"].(string); ok {
		model.CreatedAt = types.StringValue(v)
	} else {
		model.CreatedAt = types.StringNull()
	}

	return nil
}
