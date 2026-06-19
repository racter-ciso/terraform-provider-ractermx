// Copyright (c) RacterMX
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// Feature: terraform-provider-ractermx, Property 6: Composite ID round-trip
// **Validates: Requirements 6.6**
func TestProperty_ZoneRecordCompositeIDRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate valid components: positive domain ID, non-empty strings without "/"
		domainID := rapid.Int64Range(1, 1<<53).Draw(t, "domainID")

		// Non-empty strings that don't contain "/" for name and type
		nameGen := rapid.StringMatching(`[a-zA-Z0-9._@-]+`)
		typeGen := rapid.StringMatching(`[A-Z]+`)
		// Content can contain "/" so we just need it non-empty
		contentGen := rapid.StringMatching(`[a-zA-Z0-9._@:/ -]+`)

		name := nameGen.Draw(t, "name")
		recordType := typeGen.Draw(t, "recordType")
		content := contentGen.Draw(t, "content")

		// Filter out empty strings (regex + quantifier should prevent this, but be safe)
		if name == "" || recordType == "" || content == "" {
			t.Skip("empty component generated")
		}

		// Format the composite ID
		formatted := FormatZoneRecordID(domainID, name, recordType, content)

		// Verify the formatted string uses "/" as separator
		if !strings.Contains(formatted, "/") {
			t.Fatalf("formatted ID should contain '/' separator, got: %q", formatted)
		}

		// Parse it back
		parsedDomainID, parsedName, parsedType, parsedContent, err := ParseZoneRecordID(formatted)
		if err != nil {
			t.Fatalf("ParseZoneRecordID(%q) returned error: %v", formatted, err)
		}

		// Verify round-trip yields original components
		if parsedDomainID != domainID {
			t.Fatalf("domainID mismatch: got %d, want %d", parsedDomainID, domainID)
		}
		if parsedName != name {
			t.Fatalf("name mismatch: got %q, want %q", parsedName, name)
		}
		if parsedType != recordType {
			t.Fatalf("recordType mismatch: got %q, want %q", parsedType, recordType)
		}
		if parsedContent != content {
			t.Fatalf("content mismatch: got %q, want %q", parsedContent, content)
		}
	})
}
