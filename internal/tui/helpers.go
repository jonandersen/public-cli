package tui

import (
	"fmt"
	"strconv"
	"strings"
)

// formatGainLoss formats a gain/loss value with +/- prefix.
func formatGainLoss(value string) string {
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

// formatVolume formats a volume number with thousand separators.
func formatVolume(vol string) string {
	if vol == "" || vol == "0" {
		return "-"
	}

	// Parse as int64
	v, err := strconv.ParseInt(vol, 10, 64)
	if err != nil {
		return vol
	}

	if v == 0 {
		return "-"
	}

	// Format with thousand separators
	str := strconv.FormatInt(v, 10)
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
