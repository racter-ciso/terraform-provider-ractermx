// Copyright (c) RacterMX
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"fmt"
	"strconv"
	"strings"
)

// FormatZoneRecordID formats a composite ID for a zone record.
// Format: {domain_id}/{name}/{type}/{content}
func FormatZoneRecordID(domainID int64, name, recordType, content string) string {
	return fmt.Sprintf("%d/%s/%s/%s", domainID, name, recordType, content)
}

// ParseZoneRecordID parses a composite zone record ID into its components.
// The ID format is {domain_id}/{name}/{type}/{content}.
// Content may contain "/" characters, so we split into at most 4 parts.
func ParseZoneRecordID(id string) (domainID int64, name, recordType, content string, err error) {
	parts := strings.SplitN(id, "/", 4)
	if len(parts) != 4 {
		err = fmt.Errorf("expected format {domain_id}/{name}/{type}/{content}, got: %q", id)
		return
	}

	domainID, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		err = fmt.Errorf("invalid domain_id %q: %w", parts[0], err)
		return
	}

	name = parts[1]
	recordType = parts[2]
	content = parts[3]

	if name == "" {
		err = fmt.Errorf("name component cannot be empty in composite ID: %q", id)
		return
	}
	if recordType == "" {
		err = fmt.Errorf("type component cannot be empty in composite ID: %q", id)
		return
	}
	if content == "" {
		err = fmt.Errorf("content component cannot be empty in composite ID: %q", id)
		return
	}

	return
}

// FormatTagAssignmentID formats a composite ID for a domain tag assignment.
// Format: {domain_id}/{tag_id}
func FormatTagAssignmentID(domainID, tagID int64) string {
	return fmt.Sprintf("%d/%d", domainID, tagID)
}

// ParseTagAssignmentID parses a composite tag assignment ID into its components.
// The ID format is {domain_id}/{tag_id}.
func ParseTagAssignmentID(id string) (domainID, tagID int64, err error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		err = fmt.Errorf("expected format {domain_id}/{tag_id}, got: %q", id)
		return
	}

	domainID, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		err = fmt.Errorf("invalid domain_id %q: %w", parts[0], err)
		return
	}

	tagID, err = strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		err = fmt.Errorf("invalid tag_id %q: %w", parts[1], err)
		return
	}

	return
}

// FormatCheckOverrideID formats a composite ID for a check override.
// Format: {domain_id}/{check_id}
func FormatCheckOverrideID(domainID int64, checkID string) string {
	return fmt.Sprintf("%d/%s", domainID, checkID)
}

// ParseCheckOverrideID parses a composite check override ID into its components.
// The ID format is {domain_id}/{check_id}.
func ParseCheckOverrideID(id string) (domainID int64, checkID string, err error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		err = fmt.Errorf("expected format {domain_id}/{check_id}, got: %q", id)
		return
	}

	domainID, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		err = fmt.Errorf("invalid domain_id %q: %w", parts[0], err)
		return
	}

	checkID = parts[1]
	if checkID == "" {
		err = fmt.Errorf("check_id component cannot be empty in composite ID: %q", id)
		return
	}

	return
}
