// Copyright (c) RacterMX
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-ractermx/internal/client"
)

// Ensure DomainVerificationResource satisfies the resource interface.
var _ resource.Resource = &DomainVerificationResource{}

// DomainVerificationResource implements the ractermx_domain_verification resource.
// This is an action resource: Create triggers DNS verification, Read refreshes
// verification status, and Delete is a no-op. Re-verification is done by
// tainting the resource and re-applying.
type DomainVerificationResource struct {
	client *client.Client
}

// DomainVerificationResourceModel maps the resource schema to a Go struct.
type DomainVerificationResourceModel struct {
	DomainID      types.Int64 `tfsdk:"domain_id"`
	MxVerified    types.Bool  `tfsdk:"mx_verified"`
	SpfVerified   types.Bool  `tfsdk:"spf_verified"`
	DkimVerified  types.Bool  `tfsdk:"dkim_verified"`
	DmarcVerified types.Bool  `tfsdk:"dmarc_verified"`
	IsVerified    types.Bool  `tfsdk:"is_verified"`
}

// NewDomainVerificationResource returns a new resource.Resource for the domain verification resource.
func NewDomainVerificationResource() resource.Resource {
	return &DomainVerificationResource{}
}

func (r *DomainVerificationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_verification"
}

func (r *DomainVerificationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Triggers DNS verification for a RacterMX domain. " +
			"This is an action resource: creating it triggers verification, and reading it refreshes the verification status. " +
			"To re-run verification, taint this resource and re-apply.",
		Attributes: map[string]schema.Attribute{
			"domain_id": schema.Int64Attribute{
				Description: "The numeric ID of the domain to verify. Changing this forces a new resource.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
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
			"is_verified": schema.BoolAttribute{
				Description: "Whether the domain is fully verified (all DNS checks pass).",
				Computed:    true,
			},
		},
	}
}

func (r *DomainVerificationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DomainVerificationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DomainVerificationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := strconv.FormatInt(plan.DomainID.ValueInt64(), 10)

	// Trigger DNS verification.
	_, err := r.client.Post(ctx, "/domains/"+domainID+"/verify-dns", nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Triggering Domain Verification",
			"Could not trigger DNS verification for domain ID "+domainID+": "+err.Error(),
		)
		return
	}

	// Read the domain to get the current verification status.
	var state DomainVerificationResourceModel
	state.DomainID = plan.DomainID

	if diags := r.readVerificationStatus(ctx, domainID, &state); diags != nil {
		resp.Diagnostics.AddError(diags.Summary, diags.Detail)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DomainVerificationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DomainVerificationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := strconv.FormatInt(state.DomainID.ValueInt64(), 10)

	result, err := r.client.Get(ctx, "/domains/"+domainID, true)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Domain Verification Status",
			"Could not read domain ID "+domainID+": "+err.Error(),
		)
		return
	}

	// nil result means 404 — domain was deleted out-of-band.
	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	var refreshed DomainVerificationResourceModel
	refreshed.DomainID = state.DomainID

	if diags := parseVerificationResponse(result, &refreshed); diags != nil {
		resp.Diagnostics.AddError(diags.Summary, diags.Detail)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &refreshed)...)
}

func (r *DomainVerificationResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Update is not needed. domain_id has RequiresReplace, so any change
	// triggers destroy+create.
	resp.Diagnostics.AddError(
		"Unexpected Update",
		"Domain verification resource does not support in-place updates. This is a bug in the provider.",
	)
}

func (r *DomainVerificationResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// No-op. Verification cannot be "undone".
}

// readVerificationStatus fetches the domain and extracts verification fields.
func (r *DomainVerificationResource) readVerificationStatus(ctx context.Context, domainID string, model *DomainVerificationResourceModel) *domainDiag {
	result, err := r.client.Get(ctx, "/domains/"+domainID, false)
	if err != nil {
		return &domainDiag{
			Summary: "Error Reading Domain Verification Status",
			Detail:  "Could not read domain ID " + domainID + ": " + err.Error(),
		}
	}

	return parseVerificationResponse(result, model)
}

// parseVerificationResponse parses the API response and extracts verification fields.
func parseVerificationResponse(body []byte, model *DomainVerificationResourceModel) *domainDiag {
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return &domainDiag{
			Summary: "Error Parsing Domain Response",
			Detail:  "Could not parse API response envelope: " + err.Error(),
		}
	}

	var data map[string]interface{}
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return &domainDiag{
			Summary: "Error Parsing Domain Data",
			Detail:  "Could not parse domain data: " + err.Error(),
		}
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
	if v, ok := data["is_verified"].(bool); ok {
		model.IsVerified = types.BoolValue(v)
	} else {
		model.IsVerified = types.BoolNull()
	}

	return nil
}
