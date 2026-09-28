package providers

import (
	"math"
	"strconv"
	"strings"
)

func formatUsageAmount(value float64) string {
	// Choose the unit from the displayed integer, but round k directly from
	// the original value to avoid double rounding (e.g. 10499.7 stays 10k).
	if math.RoundToEven(value) >= 10000 {
		return strconv.FormatFloat(value/1000, 'f', 0, 64) + "k"
	}
	return strconv.FormatFloat(value, 'f', 0, 64)
}

func formatFullUsageAmount(value float64) string {
	return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(value, 'f', 2, 64), "0"), ".")
}

// formatUsageDetails provides remaining/limit amounts for the row and dot menu.
func formatUsageDetails(used, remaining, limit float64) (detail string, amounts *DisplayUsageAmounts) {
	return formatUsageAmount(remaining) + "/" + formatUsageAmount(limit),
		&DisplayUsageAmounts{Used: formatFullUsageAmount(used), Limit: formatFullUsageAmount(limit), Remaining: formatFullUsageAmount(remaining)}
}
