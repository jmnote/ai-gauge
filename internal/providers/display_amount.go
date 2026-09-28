package providers

import (
	"strconv"
	"strings"
)

func formatUsageAmount(value float64) string {
	if value >= 10000 {
		return strconv.FormatFloat(value/1000, 'f', 0, 64) + "k"
	}
	return formatFullUsageAmount(value)
}

func formatFullUsageAmount(value float64) string {
	return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(value, 'f', 2, 64), "0"), ".")
}

// formatUsageDetails provides remaining/limit amounts for the row and dot menu.
func formatUsageDetails(remaining, limit float64) (detail, full string) {
	return formatUsageAmount(remaining) + "/" + formatUsageAmount(limit),
		formatFullUsageAmount(remaining) + "/" + formatFullUsageAmount(limit)
}
