package providers

import "testing"

func TestUsageDetailFormatting(t *testing.T) {
	for _, tc := range []struct {
		remaining, limit float64
		detail, full     string
	}{
		{36.75, 1000, "36.75/1000", "36.75/1000"},
		{0.04, 1000, "0.04/1000", "0.04/1000"},
		{9999, 9999, "9999/9999", "9999/9999"},
		{10000, 26900, "10k/27k", "10000/26900"},
		{3630.43, 25000, "3630.43/25k", "3630.43/25000"},
		{0.04, 25000, "0.04/25k", "0.04/25000"},
		{26900.43, 30000, "27k/30k", "26900.43/30000"},
	} {
		detail, amounts := formatUsageDetails(0, tc.remaining, tc.limit)
		full := amounts.Remaining + "/" + amounts.Limit
		if detail != tc.detail || full != tc.full {
			t.Errorf("formatUsageDetails(%g, %g) = (%q, %q), want (%q, %q)", tc.remaining, tc.limit, detail, full, tc.detail, tc.full)
		}
	}
}
