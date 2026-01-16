// Package publicapi provides shared utilities for the Public.com API client.
package publicapi

import (
	"fmt"
	"strconv"
	"strings"
)

// FormatGainLoss formats a gain/loss value with +/- prefix.
// Returns "$0.00" for empty, zero, or invalid values.
func FormatGainLoss(value string) string {
	if value == "" || value == "0" || value == "0.00" {
		return "$0.00"
	}
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return "$" + value
	}
	if f >= 0 {
		return fmt.Sprintf("+$%.2f", f)
	}
	return fmt.Sprintf("-$%.2f", -f)
}

// FormatVolume formats a volume number with thousand separators.
// Returns "-" for zero values.
func FormatVolume(vol int64) string {
	if vol == 0 {
		return "-"
	}

	str := strconv.FormatInt(vol, 10)
	n := len(str)
	if n <= 3 {
		return str
	}

	var result strings.Builder
	remainder := n % 3
	if remainder > 0 {
		result.WriteString(str[:remainder])
		if n > remainder {
			result.WriteString(",")
		}
	}

	for i := remainder; i < n; i += 3 {
		result.WriteString(str[i : i+3])
		if i+3 < n {
			result.WriteString(",")
		}
	}

	return result.String()
}
