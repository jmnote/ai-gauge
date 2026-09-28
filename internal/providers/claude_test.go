package providers

import (
	"math"
	"strings"
	"testing"
)

func TestClaudeExtraUsage(t *testing.T) {
	fixture := capturedUsageFixture(t, "usage_claude.monthly.json")
	usage := connectedClaude(t, fixture)
	display := usage.ToDisplay()
	if display.Status != StatusConnected || display.Error != "" || len(display.Groups) != 1 || len(display.Groups[0].Buckets) != 1 {
		t.Fatalf("monthly-only display = %#v", display)
	}
	bucket := display.Groups[0].Buckets[0]
	if bucket.Label != "mo" || bucket.Detail != "1221/2000" || bucket.DetailHover != "1221/2000" || math.Abs(bucket.Remaining-38.95) > 1e-9 || bucket.ResetTime != "" {
		t.Fatalf("monthly bucket = %#v", bucket)
	}
	regular := connectedClaude(t, readFixture(t, "claude-usage.json"))
	regular.ExtraUsage = usage.ExtraUsage
	if got := regular.ToDisplay(); got.Error != "" || len(got.Groups[0].Buckets) != 4 || got.Groups[0].Buckets[3].Label != "mo" {
		t.Fatalf("regular and extra usage = %#v", got)
	}
	for _, state := range []string{"disabled", "absent"} {
		t.Run(state, func(t *testing.T) {
			current := regular
			if state == "disabled" {
				current.ExtraUsage = &claudeExtraUsage{IsEnabled: false}
			} else {
				current.ExtraUsage = nil
			}
			got := current.ToDisplay()
			if got.Error != "" || len(got.Groups[0].Buckets) != 3 {
				t.Fatalf("display = %#v", got)
			}
			current.FiveHour = claudeUsageWindow{}
			current.SevenDay = claudeUsageWindow{}
			current.SevenDayOpus = nil
			if got := current.ToDisplay(); got.Error == "" {
				t.Fatal("expected no usage data when all limits are absent or disabled")
			}
		})
	}
	for _, tc := range []struct{ name, from, to string }{
		{"missing limit", `"monthly_limit": 2000`, `"monthly_limit": null`},
		{"negative used", `"used_credits": 1221`, `"used_credits": -1`},
		{"missing utilization", `"utilization": 61.050000000000004`, `"utilization": null`},
		{"invalid utilization", `"utilization": 61.050000000000004`, `"utilization": 101`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := connectedClaude(t, []byte(strings.Replace(string(fixture), tc.from, tc.to, 1))).ToDisplay()
			if got.Error == "" {
				t.Fatal("expected validation error")
			}
		})
	}
}
