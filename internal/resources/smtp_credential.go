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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-ractermx/internal/client"
)

// Ensure SmtpCredentialResource satisfies the resource interfaces.
var (
	_ resource.Resource                = &SmtpCredentialResource{}
	_ resource.ResourceWithImportState = &SmtpCredentialResource{}
)

// SmtpCredentialResource implements the ractermx_smtp_credential resource.
type SmtpCredentialResource struct {
	client *client.Client
}

// SmtpCredentialResourceModel maps the resource schema to a Go struct.
type SmtpCredentialResourceModel struct {
	ID                    types.Int64  `tfsdk:"id"`
	DomainID              types.Int64  `tfsdk:"domain_id"`
	DailyLimit            types.Int64  `tfsdk:"daily_limit"`
	AnonymousReplyEnabled types.Bool   `tfsdk:"anonymous_reply_enabled"`
	ProxyDomainID         types.Int64  `tfsdk:"proxy_domain_id"`
	// Computed
	Username   types.String `tfsdk:"username"`
	Password   types.String `tfsdk:"password"`
	SmtpConfig types.Object `tfsdk:"smtp_config"`
}

// smtpConfigAttrTypes defines the attribute types for the smtp_config nested object.
var smtpConfigAttrTypes = map[string]attr.Type{
	"host":       types.StringType,
	"port":       types.Int64Type,
	"encryption": types.StringType,
}

// NewSmtpCredentialResource returns a new resource.Resource for the SMTP credential resource.
func NewSmtpCredentialResource() resource.Resource {
	return &SmtpCredentialResource{}
}

func (r *SmtpCredentialResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_smtp_credential"
}

func (r *SmtpCredentialResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a RacterMX SMTP credential.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The numeric ID of the SMTP credential.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"domain_id": schema.Int64Attribute{
				Description: "The numeric ID of the domain. Changing this forces a new resource.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"daily_limit": schema.Int64Attribute{
				Description: "Daily send limit (1-100000, default 1000). Changing this forces a new resource.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"anonymous_reply_enabled": schema.BoolAttribute{
				Description: "Whether anonymous reply is enabled. Changing this forces a new resource.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"proxy_domain_id": schema.Int64Attribute{
				Description: "The proxy domain ID for anonymous replies. Changing this forces a new resource.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
					int64planmodifier.UseStateForUnknown(),
				},
			},
			// Computed attributes
			"username": schema.StringAttribute{
				Description: "The SMTP username.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"password": schema.StringAttribute{
				Description: "The SMTP password. Only available on creation; will be empty after import.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"smtp_config": schema.SingleNestedAttribute{
				Description: "SMTP server configuration.",
				Computed:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"host": schema.StringAttribute{
						Description: "SMTP server hostname.",
						Computed:    true,
					},
					"port": schema.Int64Attribute{
						Description: "SMTP server port.",
						Computed:    true,
					},
					"encryption": schema.StringAttribute{
						Description: "SMTP encryption type.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (r *SmtpCredentialResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SmtpCredentialResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SmtpCredentialResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := strconv.FormatInt(plan.DomainID.ValueInt64(), 10)

	body := map[string]interface{}{}

	if !plan.DailyLimit.IsNull() && !plan.DailyLimit.IsUnknown() {
		body["daily_limit"] = plan.DailyLimit.ValueInt64()
	}
	if !plan.AnonymousReplyEnabled.IsNull() && !plan.AnonymousReplyEnabled.IsUnknown() {
		body["anonymous_reply_enabled"] = plan.AnonymousReplyEnabled.ValueBool()
	}
	if !plan.ProxyDomainID.IsNull() && !plan.ProxyDomainID.IsUnknown() {
		body["proxy_domain_id"] = plan.ProxyDomainID.ValueInt64()
	}

	result, err := r.client.Post(ctx, "/domains/"+domainID+"/smtp-credentials", body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating SMTP Credential",
			"Could not create SMTP credential, unexpected error: "+err.Error(),
		)
		return
	}

	var state SmtpCredentialResourceModel
	if diags := parseSmtpCredentialResponse(result, &state); diags != nil {
		resp.Diagnostics.AddError(diags.Summary, diags.Detail)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SmtpCredentialResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SmtpCredentialResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	credentialID := state.ID.ValueInt64()
	domainID := strconv.FormatInt(state.DomainID.ValueInt64(), 10)

	// SMTP credentials have no individual GET endpoint. List all for the domain and match by ID.
	result, err := r.client.Get(ctx, "/domains/"+domainID+"/smtp-credentials", false)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading SMTP Credential",
			"Could not read SMTP credentials: "+err.Error(),
		)
		return
	}

	credential, err := findSmtpCredentialByID(result, credentialID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing SMTP Credentials",
			"Could not parse SMTP credentials response: "+err.Error(),
		)
		return
	}

	if credential == nil {
		// Credential not found — removed out-of-band.
		resp.State.RemoveResource(ctx)
		return
	}

	var refreshed SmtpCredentialResourceModel
	parseSmtpCredentialData(credential, &refreshed)

	// Preserve the password from state since the API does not return it on read.
	refreshed.Password = state.Password

	resp.Diagnostics.Append(resp.State.Set(ctx, &refreshed)...)
}

func (r *SmtpCredentialResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// No update endpoint — all configurable attribute changes force replacement via RequiresReplace().
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"SMTP credentials cannot be updated. Attribute changes force replacement.",
	)
}

func (r *SmtpCredentialResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SmtpCredentialResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	credentialID := strconv.FormatInt(state.ID.ValueInt64(), 10)

	err := r.client.Delete(ctx, "/smtp-credentials/"+credentialID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting SMTP Credential",
			"Could not delete SMTP credential ID "+credentialID+": "+err.Error(),
		)
		return
	}
}

func (r *SmtpCredentialResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected a numeric SMTP credential ID, got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

// smtpCredentialDiag is a simple struct for returning parse errors.
type smtpCredentialDiag struct {
	Summary string
	Detail  string
}

// findSmtpCredentialByID searches the SMTP credentials list response for a credential matching the given ID.
func findSmtpCredentialByID(body []byte, credentialID int64) (map[string]interface{}, error) {
	var envelope struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("could not parse SMTP credentials response: %w", err)
	}

	for _, raw := range envelope.Data {
		var cred map[string]interface{}
		if err := json.Unmarshal(raw, &cred); err != nil {
			continue
		}

		if id, ok := cred["id"].(float64); ok && int64(id) == credentialID {
			return cred, nil
		}
	}

	return nil, nil
}

// parseSmtpCredentialResponse parses the API response envelope {"data": {...}} and maps to a SmtpCredentialResourceModel.
func parseSmtpCredentialResponse(body []byte, model *SmtpCredentialResourceModel) *smtpCredentialDiag {
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return &smtpCredentialDiag{
			Summary: "Error Parsing SMTP Credential Response",
			Detail:  "Could not parse API response envelope: " + err.Error(),
		}
	}

	var data map[string]interface{}
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return &smtpCredentialDiag{
			Summary: "Error Parsing SMTP Credential Data",
			Detail:  "Could not parse SMTP credential data: " + err.Error(),
		}
	}

	parseSmtpCredentialData(data, model)
	return nil
}

// parseSmtpCredentialData maps an SMTP credential data object to a SmtpCredentialResourceModel.
func parseSmtpCredentialData(data map[string]interface{}, model *SmtpCredentialResourceModel) {
	if v, ok := data["id"].(float64); ok {
		model.ID = types.Int64Value(int64(v))
	}
	if v, ok := data["domain_id"].(float64); ok {
		model.DomainID = types.Int64Value(int64(v))
	}
	if v, ok := data["daily_limit"].(float64); ok {
		model.DailyLimit = types.Int64Value(int64(v))
	} else {
		model.DailyLimit = types.Int64Null()
	}
	if v, ok := data["anonymous_reply_enabled"].(bool); ok {
		model.AnonymousReplyEnabled = types.BoolValue(v)
	} else {
		model.AnonymousReplyEnabled = types.BoolNull()
	}
	if v, ok := data["proxy_domain_id"].(float64); ok {
		model.ProxyDomainID = types.Int64Value(int64(v))
	} else {
		model.ProxyDomainID = types.Int64Null()
	}
	if v, ok := data["username"].(string); ok {
		model.Username = types.StringValue(v)
	} else {
		model.Username = types.StringNull()
	}

	// Password is only returned on create.
	if v, ok := data["password"].(string); ok && v != "" {
		model.Password = types.StringValue(v)
	} else {
		model.Password = types.StringNull()
	}

	// Parse smtp_config nested object.
	if configRaw, ok := data["smtp_config"].(map[string]interface{}); ok {
		attrs := map[string]attr.Value{
			"host":       types.StringNull(),
			"port":       types.Int64Null(),
			"encryption": types.StringNull(),
		}
		if v, ok := configRaw["host"].(string); ok {
			attrs["host"] = types.StringValue(v)
		}
		if v, ok := configRaw["port"].(float64); ok {
			attrs["port"] = types.Int64Value(int64(v))
		}
		if v, ok := configRaw["encryption"].(string); ok {
			attrs["encryption"] = types.StringValue(v)
		}
		obj, diags := types.ObjectValue(smtpConfigAttrTypes, attrs)
		if !diags.HasError() {
			model.SmtpConfig = obj
		} else {
			model.SmtpConfig = types.ObjectNull(smtpConfigAttrTypes)
		}
	} else {
		model.SmtpConfig = types.ObjectNull(smtpConfigAttrTypes)
	}
}
