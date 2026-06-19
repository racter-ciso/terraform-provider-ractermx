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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-ractermx/internal/client"
)

// Ensure BlocklistEntryResource satisfies the resource interfaces.
var (
	_ resource.Resource                = &BlocklistEntryResource{}
	_ resource.ResourceWithImportState = &BlocklistEntryResource{}
)

// BlocklistEntryResource implements the ractermx_blocklist_entry resource.
type BlocklistEntryResource struct {
	client *client.Client
}

// BlocklistEntryResourceModel maps the resource schema to a Go struct.
type BlocklistEntryResourceModel struct {
	ID        types.Int64  `tfsdk:"id"`
	Pattern   types.String `tfsdk:"pattern"`
	CreatedAt types.String `tfsdk:"created_at"`
}

// NewBlocklistEntryResource returns a new resource.Resource for the blocklist entry resource.
func NewBlocklistEntryResource() resource.Resource {
	return &BlocklistEntryResource{}
}

func (r *BlocklistEntryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blocklist_entry"
}

func (r *BlocklistEntryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a RacterMX sender blocklist entry.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The numeric ID of the blocklist entry.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"pattern": schema.StringAttribute{
				Description: "The sender email pattern to block (e.g., 'spam@example.com' or '*@spam.com'). Changing this forces a new resource.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the blocklist entry was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *BlocklistEntryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BlocklistEntryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BlocklistEntryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"pattern": plan.Pattern.ValueString(),
	}

	result, err := r.client.Post(ctx, "/blocklist", body)
	if err != nil {
		// Check for 409 Conflict and provide a clear error message.
		if apiErr, ok := err.(*client.APIError); ok && apiErr.StatusCode == 409 {
			resp.Diagnostics.AddError(
				"Blocklist Entry Already Exists",
				fmt.Sprintf("A blocklist entry for pattern '%s' already exists: %s", plan.Pattern.ValueString(), apiErr.Message),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error Creating Blocklist Entry",
			"Could not create blocklist entry, unexpected error: "+err.Error(),
		)
		return
	}

	// Parse the API response envelope: {"data": {...}}
	var state BlocklistEntryResourceModel
	if diags := parseBlocklistEntryResponse(result, &state); diags != nil {
		resp.Diagnostics.AddError(diags.Summary, diags.Detail)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *BlocklistEntryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BlocklistEntryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	entryID := state.ID.ValueInt64()

	// Blocklist entries have no individual GET endpoint. List all and match by ID.
	result, err := r.client.Get(ctx, "/blocklist", false)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Blocklist Entry",
			"Could not read blocklist entries: "+err.Error(),
		)
		return
	}

	entry, err := findBlocklistEntryByID(result, entryID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Blocklist Entries",
			"Could not parse blocklist entries response: "+err.Error(),
		)
		return
	}

	if entry == nil {
		// Entry not found — removed out-of-band.
		resp.State.RemoveResource(ctx)
		return
	}

	var refreshed BlocklistEntryResourceModel
	parseBlocklistEntryData(entry, &refreshed)

	resp.Diagnostics.Append(resp.State.Set(ctx, &refreshed)...)
}

func (r *BlocklistEntryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// No update endpoint — pattern changes force replacement via RequiresReplace().
	// This method should never be called, but is required by the interface.
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Blocklist entries cannot be updated. Pattern changes force replacement.",
	)
}

func (r *BlocklistEntryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BlocklistEntryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	entryID := strconv.FormatInt(state.ID.ValueInt64(), 10)

	err := r.client.Delete(ctx, "/blocklist/"+entryID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Blocklist Entry",
			"Could not delete blocklist entry ID "+entryID+": "+err.Error(),
		)
		return
	}
}

func (r *BlocklistEntryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID as an integer.
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected a numeric blocklist entry ID, got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

// blocklistEntryDiag is a simple struct for returning parse errors.
type blocklistEntryDiag struct {
	Summary string
	Detail  string
}

// findBlocklistEntryByID searches the blocklist response for an entry matching the given ID.
// Returns nil if no match is found.
func findBlocklistEntryByID(body []byte, entryID int64) (map[string]interface{}, error) {
	var envelope struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("could not parse blocklist response: %w", err)
	}

	for _, raw := range envelope.Data {
		var entry map[string]interface{}
		if err := json.Unmarshal(raw, &entry); err != nil {
			continue
		}

		if id, ok := entry["id"].(float64); ok && int64(id) == entryID {
			return entry, nil
		}
	}

	return nil, nil
}

// parseBlocklistEntryResponse parses the API response envelope {"data": {...}} and maps to a BlocklistEntryResourceModel.
func parseBlocklistEntryResponse(body []byte, model *BlocklistEntryResourceModel) *blocklistEntryDiag {
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return &blocklistEntryDiag{
			Summary: "Error Parsing Blocklist Entry Response",
			Detail:  "Could not parse API response envelope: " + err.Error(),
		}
	}

	var data map[string]interface{}
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return &blocklistEntryDiag{
			Summary: "Error Parsing Blocklist Entry Data",
			Detail:  "Could not parse blocklist entry data: " + err.Error(),
		}
	}

	parseBlocklistEntryData(data, model)
	return nil
}

// parseBlocklistEntryData maps a blocklist entry data object to a BlocklistEntryResourceModel.
func parseBlocklistEntryData(data map[string]interface{}, model *BlocklistEntryResourceModel) {
	if v, ok := data["id"].(float64); ok {
		model.ID = types.Int64Value(int64(v))
	}
	if v, ok := data["pattern"].(string); ok {
		model.Pattern = types.StringValue(v)
	}
	if v, ok := data["created_at"].(string); ok {
		model.CreatedAt = types.StringValue(v)
	} else {
		model.CreatedAt = types.StringNull()
	}
}
