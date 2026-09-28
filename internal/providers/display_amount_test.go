package providers

import "testing"

func TestUsageDetailFormatting(t *testing.T) {
	for _, tc := range []struct {
		remaining, limit float64
		detail, full     string
	}{
		{36.75, 1000, "37/1000", "36.75/1000"},
		{0.04, 1000, "0/1000", "0.04/1000"},
		{9999, 9999, "9999/9999", "9999/9999"},
		{9999.49, 25000, "9999/25k", "9999.49/25000"},
		{9999.5, 25000, "10k/25k", "9999.5/25000"},
		{9999.7, 25000, "10k/25k", "9999.7/25000"},
		{9999.99, 25000, "10k/25k", "9999.99/25000"},
		{1, 9999.7, "1/10k", "1/9999.7"},
		{10499.7, 25000, "10k/25k", "10499.7/25000"},
		{10000, 26900, "10k/27k", "10000/26900"},
		{3630.43, 25000, "3630/25k", "3630.43/25000"},
		{0.04, 25000, "0/25k", "0.04/25000"},
		{26900.43, 30000, "27k/30k", "26900.43/30000"},
	} {
		detail, amounts := formatUsageDetails(0, tc.remaining, tc.limit)
		full := amounts.Remaining + "/" + amounts.Limit
		if detail != tc.detail || full != tc.full {
			t.Errorf("formatUsageDetails(%g, %g) = (%q, %q), want (%q, %q)", tc.remaining, tc.limit, detail, full, tc.detail, tc.full)
		}
	}
}
