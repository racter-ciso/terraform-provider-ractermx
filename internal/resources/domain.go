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

// Ensure DomainResource satisfies the resource interfaces.
var (
	_ resource.Resource                = &DomainResource{}
	_ resource.ResourceWithImportState = &DomainResource{}
)

// DomainResource implements the ractermx_domain resource.
type DomainResource struct {
	client *client.Client
}

// DomainResourceModel maps the resource schema to a Go struct.
type DomainResourceModel struct {
	ID                types.Int64  `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	OrganizationID    types.Int64  `tfsdk:"organization_id"`
	IsForwarding      types.Bool   `tfsdk:"is_forwarding"`
	IsMonitored       types.Bool   `tfsdk:"is_monitored"`
	IsHosted          types.Bool   `tfsdk:"is_hosted"`
	DnsMode           types.String `tfsdk:"dns_mode"`
	CatchAllEnabled   types.Bool   `tfsdk:"catch_all_enabled"`
	CatchAllForwardTo types.String `tfsdk:"catch_all_forward_to"`
	MaxAliases                  types.Int64  `tfsdk:"max_aliases"`
	UnsubscribeRewritingEnabled types.Bool   `tfsdk:"unsubscribe_rewriting_enabled"`
	// Computed
	IsActive          types.Bool   `tfsdk:"is_active"`
	IsVerified        types.Bool   `tfsdk:"is_verified"`
	VerificationToken types.String `tfsdk:"verification_token"`
	MxVerified        types.Bool   `tfsdk:"mx_verified"`
	SpfVerified       types.Bool   `tfsdk:"spf_verified"`
	DkimVerified      types.Bool   `tfsdk:"dkim_verified"`
	DmarcVerified     types.Bool   `tfsdk:"dmarc_verified"`
	LastVerifiedAt    types.String `tfsdk:"last_verified_at"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}

// NewDomainResource returns a new resource.Resource for the domain resource.
func NewDomainResource() resource.Resource {
	return &DomainResource{}
}

func (r *DomainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (r *DomainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a RacterMX domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The numeric ID of the domain.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The domain name (e.g., example.com). Changing this forces a new resource.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"organization_id": schema.Int64Attribute{
				Description: "The organization ID to assign this domain to.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"is_forwarding": schema.BoolAttribute{
				Description: "Whether email forwarding is enabled for this domain.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"is_monitored": schema.BoolAttribute{
				Description: "Whether monitoring is enabled for this domain.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"is_hosted": schema.BoolAttribute{
				Description: "Whether DNS hosting is enabled for this domain.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"dns_mode": schema.StringAttribute{
				Description: "The DNS mode for this domain. One of: scan_only, mx_forwarding, dns_hosted.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"catch_all_enabled": schema.BoolAttribute{
				Description: "Whether catch-all forwarding is enabled.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"catch_all_forward_to": schema.StringAttribute{
				Description: "The email address to forward catch-all mail to.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"max_aliases": schema.Int64Attribute{
				Description: "Maximum number of aliases allowed (1-1000, default 100).",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"unsubscribe_rewriting_enabled": schema.BoolAttribute{
				Description: "Whether unsubscribe link rewriting is enabled for this domain.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			// Computed attributes
			"is_active": schema.BoolAttribute{
				Description: "Whether the domain is active.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"is_verified": schema.BoolAttribute{
				Description: "Whether the domain has been verified.",
				Computed:    true,
			},
			"verification_token": schema.StringAttribute{
				Description: "The verification token for this domain.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mx_verified": schema.BoolAttribute{
				Description: "Whether MX records are verified.",
				Computed:    true,
			},
			"spf_verified": schema.BoolAttribute{
				Description: "Whether SPF records are verified.",
				Computed:    true,
			},
			"dkim_verified": schema.BoolAttribute{
				Description: "Whether DKIM records are verified.",
				Computed:    true,
			},
			"dmarc_verified": schema.BoolAttribute{
				Description: "Whether DMARC records are verified.",
				Computed:    true,
			},
			"last_verified_at": schema.StringAttribute{
				Description: "The timestamp of the last verification check.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the domain was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the domain was last updated.",
				Computed:    true,
			},
		},
	}
}

func (r *DomainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build request body from the plan.
	body := map[string]interface{}{
		"name": plan.Name.ValueString(),
	}

	if !plan.OrganizationID.IsNull() && !plan.OrganizationID.IsUnknown() {
		body["organization_id"] = plan.OrganizationID.ValueInt64()
	}
	if !plan.IsForwarding.IsNull() && !plan.IsForwarding.IsUnknown() {
		body["is_forwarding"] = plan.IsForwarding.ValueBool()
	}
	if !plan.IsMonitored.IsNull() && !plan.IsMonitored.IsUnknown() {
		body["is_monitored"] = plan.IsMonitored.ValueBool()
	}
	if !plan.IsHosted.IsNull() && !plan.IsHosted.IsUnknown() {
		body["is_hosted"] = plan.IsHosted.ValueBool()
	}
	if !plan.DnsMode.IsNull() && !plan.DnsMode.IsUnknown() {
		body["dns_mode"] = plan.DnsMode.ValueString()
	}
	if !plan.CatchAllEnabled.IsNull() && !plan.CatchAllEnabled.IsUnknown() {
		body["catch_all_enabled"] = plan.CatchAllEnabled.ValueBool()
	}
	if !plan.CatchAllForwardTo.IsNull() && !plan.CatchAllForwardTo.IsUnknown() {
		body["catch_all_forward_to"] = plan.CatchAllForwardTo.ValueString()
	}
	if !plan.MaxAliases.IsNull() && !plan.MaxAliases.IsUnknown() {
		body["max_aliases"] = plan.MaxAliases.ValueInt64()
	}
	if !plan.UnsubscribeRewritingEnabled.IsNull() && !plan.UnsubscribeRewritingEnabled.IsUnknown() {
		body["unsubscribe_rewriting_enabled"] = plan.UnsubscribeRewritingEnabled.ValueBool()
	}

	result, err := r.client.Post(ctx, "/domains", body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Domain",
			"Could not create domain, unexpected error: "+err.Error(),
		)
		return
	}

	// Parse the API response envelope: {"data": {...}}
	var state DomainResourceModel
	if diags := parseDomainResponse(result, &state); diags != nil {
		resp.Diagnostics.AddError(diags.Summary, diags.Detail)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := strconv.FormatInt(state.ID.ValueInt64(), 10)
	result, err := r.client.Get(ctx, "/domains/"+domainID, true)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Domain",
			"Could not read domain ID "+domainID+": "+err.Error(),
		)
		return
	}

	// nil result means 404 — resource was deleted out-of-band.
	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	var refreshed DomainResourceModel
	if diags := parseDomainResponse(result, &refreshed); diags != nil {
		resp.Diagnostics.AddError(diags.Summary, diags.Detail)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &refreshed)...)
}

func (r *DomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state DomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := strconv.FormatInt(state.ID.ValueInt64(), 10)

	// Build request body with only changed fields.
	body := map[string]interface{}{}

	if !plan.OrganizationID.Equal(state.OrganizationID) {
		if plan.OrganizationID.IsNull() {
			body["organization_id"] = nil
		} else {
			body["organization_id"] = plan.OrganizationID.ValueInt64()
		}
	}
	if !plan.IsForwarding.Equal(state.IsForwarding) {
		body["is_forwarding"] = plan.IsForwarding.ValueBool()
	}
	if !plan.IsMonitored.Equal(state.IsMonitored) {
		body["is_monitored"] = plan.IsMonitored.ValueBool()
	}
	if !plan.IsHosted.Equal(state.IsHosted) {
		body["is_hosted"] = plan.IsHosted.ValueBool()
	}
	if !plan.DnsMode.Equal(state.DnsMode) {
		body["dns_mode"] = plan.DnsMode.ValueString()
	}
	if !plan.CatchAllEnabled.Equal(state.CatchAllEnabled) {
		body["catch_all_enabled"] = plan.CatchAllEnabled.ValueBool()
	}
	if !plan.CatchAllForwardTo.Equal(state.CatchAllForwardTo) {
		if plan.CatchAllForwardTo.IsNull() {
			body["catch_all_forward_to"] = nil
		} else {
			body["catch_all_forward_to"] = plan.CatchAllForwardTo.ValueString()
		}
	}
	if !plan.MaxAliases.Equal(state.MaxAliases) {
		body["max_aliases"] = plan.MaxAliases.ValueInt64()
	}
	if !plan.UnsubscribeRewritingEnabled.Equal(state.UnsubscribeRewritingEnabled) {
		body["unsubscribe_rewriting_enabled"] = plan.UnsubscribeRewritingEnabled.ValueBool()
	}

	result, err := r.client.Patch(ctx, "/domains/"+domainID, body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Domain",
			"Could not update domain ID "+domainID+": "+err.Error(),
		)
		return
	}

	var updated DomainResourceModel
	if diags := parseDomainResponse(result, &updated); diags != nil {
		resp.Diagnostics.AddError(diags.Summary, diags.Detail)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
}

func (r *DomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := strconv.FormatInt(state.ID.ValueInt64(), 10)

	err := r.client.Delete(ctx, "/domains/"+domainID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Domain",
			"Could not delete domain ID "+domainID+": "+err.Error(),
		)
		return
	}
}

func (r *DomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID as an integer.
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected a numeric domain ID, got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

// domainDiag is a simple struct for returning parse errors.
type domainDiag struct {
	Summary string
	Detail  string
}

// parseDomainResponse parses the API response envelope and maps the data to a DomainResourceModel.
func parseDomainResponse(body []byte, model *DomainResourceModel) *domainDiag {
	// Parse the outer envelope: {"data": {...}}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return &domainDiag{
			Summary: "Error Parsing Domain Response",
			Detail:  "Could not parse API response envelope: " + err.Error(),
		}
	}

	// Parse the inner data object.
	var data map[string]interface{}
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return &domainDiag{
			Summary: "Error Parsing Domain Data",
			Detail:  "Could not parse domain data: " + err.Error(),
		}
	}

	// Map API response fields to the model.
	if v, ok := data["id"].(float64); ok {
		model.ID = types.Int64Value(int64(v))
	}
	if v, ok := data["name"].(string); ok {
		model.Name = types.StringValue(v)
	}
	if v, ok := data["organization_id"].(float64); ok {
		model.OrganizationID = types.Int64Value(int64(v))
	} else {
		model.OrganizationID = types.Int64Null()
	}
	if v, ok := data["is_forwarding"].(bool); ok {
		model.IsForwarding = types.BoolValue(v)
	} else {
		model.IsForwarding = types.BoolNull()
	}
	if v, ok := data["is_monitored"].(bool); ok {
		model.IsMonitored = types.BoolValue(v)
	} else {
		model.IsMonitored = types.BoolNull()
	}
	if v, ok := data["is_hosted"].(bool); ok {
		model.IsHosted = types.BoolValue(v)
	} else {
		model.IsHosted = types.BoolNull()
	}
	if v, ok := data["dns_mode"].(string); ok {
		model.DnsMode = types.StringValue(v)
	} else {
		model.DnsMode = types.StringNull()
	}
	if v, ok := data["catch_all_enabled"].(bool); ok {
		model.CatchAllEnabled = types.BoolValue(v)
	} else {
		model.CatchAllEnabled = types.BoolNull()
	}
	if v, ok := data["catch_all_forward_to"].(string); ok {
		model.CatchAllForwardTo = types.StringValue(v)
	} else {
		model.CatchAllForwardTo = types.StringNull()
	}
	if v, ok := data["max_aliases"].(float64); ok {
		model.MaxAliases = types.Int64Value(int64(v))
	} else {
		model.MaxAliases = types.Int64Null()
	}
	if v, ok := data["unsubscribe_rewriting_enabled"].(bool); ok {
		model.UnsubscribeRewritingEnabled = types.BoolValue(v)
	} else {
		model.UnsubscribeRewritingEnabled = types.BoolNull()
	}

	// Computed attributes
	if v, ok := data["is_active"].(bool); ok {
		model.IsActive = types.BoolValue(v)
	} else {
		model.IsActive = types.BoolNull()
	}
	if v, ok := data["is_verified"].(bool); ok {
		model.IsVerified = types.BoolValue(v)
	} else {
		model.IsVerified = types.BoolNull()
	}
	if v, ok := data["verification_token"].(string); ok {
		model.VerificationToken = types.StringValue(v)
	} else {
		model.VerificationToken = types.StringNull()
	}
	if v, ok := data["mx_verified"].(bool); ok {
		model.MxVerified = types.BoolValue(v)
	} else {
		model.MxVerified = types.BoolNull()
	}
	if v, ok := data["spf_verified"].(bool); ok {
		model.SpfVerified = types.BoolValue(v)
	} else {
		model.SpfVerified = types.BoolNull()
	}
	if v, ok := data["dkim_verified"].(bool); ok {
		model.DkimVerified = types.BoolValue(v)
	} else {
		model.DkimVerified = types.BoolNull()
	}
	if v, ok := data["dmarc_verified"].(bool); ok {
		model.DmarcVerified = types.BoolValue(v)
	} else {
		model.DmarcVerified = types.BoolNull()
	}
	if v, ok := data["last_verified_at"].(string); ok {
		model.LastVerifiedAt = types.StringValue(v)
	} else {
		model.LastVerifiedAt = types.StringNull()
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
