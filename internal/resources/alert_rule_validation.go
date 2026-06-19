// Copyright (c) RacterMX
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"fmt"
	"strconv"
)

// ValidateAlertRule validates the cross-field constraints for an alert rule.
// It checks that the combination of alert_type, condition, and threshold_value
// is valid according to the business rules.
func ValidateAlertRule(alertType, condition string, thresholdValue *string) error {
	switch alertType {
	case "blacklist_change":
		// Must use any_change condition and null threshold.
		if condition != "any_change" {
			return fmt.Errorf("alert_type 'blacklist_change' requires condition 'any_change', got %q", condition)
		}
		if thresholdValue != nil {
			return fmt.Errorf("alert_type 'blacklist_change' requires threshold_value to be null")
		}
		return nil

	case "deliverability_score", "security_posture":
		// Must not use any_change condition and threshold must be a valid grade.
		if condition == "any_change" {
			return fmt.Errorf("alert_type %q does not support condition 'any_change'", alertType)
		}
		if thresholdValue == nil {
			return fmt.Errorf("alert_type %q requires a threshold_value (valid grade: A, B, C, D, F)", alertType)
		}
		validGrades := map[string]bool{"A": true, "B": true, "C": true, "D": true, "F": true}
		if !validGrades[*thresholdValue] {
			return fmt.Errorf("alert_type %q requires threshold_value to be a valid grade (A, B, C, D, F), got %q", alertType, *thresholdValue)
		}
		return nil

	case "dmarc_compliance":
		// Must not use any_change condition and threshold must be integer 0-100.
		if condition == "any_change" {
			return fmt.Errorf("alert_type 'dmarc_compliance' does not support condition 'any_change'")
		}
		if thresholdValue == nil {
			return fmt.Errorf("alert_type 'dmarc_compliance' requires a threshold_value (integer 0-100)")
		}
		val, err := strconv.Atoi(*thresholdValue)
		if err != nil {
			return fmt.Errorf("alert_type 'dmarc_compliance' requires threshold_value to be an integer, got %q", *thresholdValue)
		}
		if val < 0 || val > 100 {
			return fmt.Errorf("alert_type 'dmarc_compliance' requires threshold_value between 0 and 100, got %d", val)
		}
		return nil

	default:
		return fmt.Errorf("unknown alert_type %q", alertType)
	}
}
