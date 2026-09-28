package providers

import "testing"

func TestUsageDetailFormatting(t *testing.T) {
	for _, tc := range []struct {
		used, limit   float64
		detail, hover string
	}{
		{36.75, 1000, "36.75/1000", ""},
		{0.04, 1000, "0.04/1000", ""},
		{9999, 9999, "9999/9999", ""},
		{10000, 26900, "10k/27k", "10000/26900"},
		{3630.43, 25000, "3630.43/25k", "3630.43/25000"},
		{0.04, 25000, "0.04/25k", "0.04/25000"},
		{26900.43, 30000, "27k/30k", "26900.4/30000"},
	} {
		detail, hover := formatUsageDetails(tc.used, tc.limit)
		if detail != tc.detail || hover != tc.hover {
			t.Errorf("formatUsageDetails(%g, %g) = (%q, %q), want (%q, %q)", tc.used, tc.limit, detail, hover, tc.detail, tc.hover)
		}
	}
}
