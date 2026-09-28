package providers

import (
	"strconv"
	"strings"
)

func formatUsageAmount(value float64) string {
	if value >= 10000 {
		return strconv.FormatFloat(value/1000, 'f', 0, 64) + "k"
	}
	return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(value, 'f', 2, 64), "0"), ".")
}

func formatUsageHoverAmount(value float64) string {
	return strings.TrimSuffix(strconv.FormatFloat(value, 'f', 1, 64), ".0")
}
