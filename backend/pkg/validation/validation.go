// Package validation contains small reusable input validation helpers.
package validation

import (
	"fmt"
	"strings"
)

// Required validates that a string contains a non-whitespace value.
func Required(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", field)
	}
	return nil
}

// IntegerRange validates an inclusive integer range.
func IntegerRange(field string, value, minimum, maximum int) error {
	if value < minimum || value > maximum {
		return fmt.Errorf("%s must be between %d and %d", field, minimum, maximum)
	}
	return nil
}
