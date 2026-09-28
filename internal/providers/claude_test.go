package providers

import (
	"math"
	"reflect"
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
	if bucket.Label != "mo" || bucket.Detail != "12.21/20" || bucket.DetailHover != "" || math.Abs(bucket.Remaining-38.95) > 1e-9 || bucket.ResetTime != "" {
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
		{"missing scale", `"decimal_places": 2`, `"decimal_places": null`},
		{"negative scale", `"decimal_places": 2`, `"decimal_places": -1`},
		{"overflowing scale", `"decimal_places": 2`, `"decimal_places": 309`},
		{"missing limit", `"monthly_limit": 2000`, `"monthly_limit": null`},
		{"negative used", `"used_credits": 1221`, `"used_credits": -1`},
		{"missing utilization", `"utilization": 61.050000000000004`, `"utilization": null`},
		{"negative utilization", `"utilization": 61.050000000000004`, `"utilization": -1`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			malformed := connectedClaude(t, []byte(strings.Replace(string(fixture), tc.from, tc.to, 1)))
			got := malformed.ToDisplay()
			regular := connectedClaude(t, readFixture(t, "claude-usage.json"))
			before := regular.ToDisplay()
			regular.ExtraUsage = malformed.ExtraUsage
			after := regular.ToDisplay()
			if after.Status != StatusConnected || after.Error != "" || !reflect.DeepEqual(after.Groups, before.Groups) {
				t.Fatalf("invalid optional extra usage changed valid windows: %#v", after)
			}
			if got.Error == "" {
				t.Fatal("expected no usage data for an unusable monthly-only response")
			}
		})
	}
}

func TestClaudeExtraUsageScaleAndOverage(t *testing.T) {
	fixture := capturedUsageFixture(t, "usage_claude.monthly.json")
	for _, tc := range []struct {
		name                     string
		places                   int
		used, limit, utilization float64
		detail                   string
		remaining                float64
	}{
		{"cents", 2, 1221, 2000, 61.05, "12.21/20", 38.95},
		{"whole units", 0, 1221, 2000, 61.05, "1221/2000", 38.95},
		{"tenths", 1, 1221, 2000, 61.05, "122.1/200", 38.95},
		{"overage", 2, 2050, 2000, 102.5, "20.5/20", 0},
		{"zero limit", 2, 0, 0, 0, "0/0", 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			usage := connectedClaude(t, fixture)
			extra := usage.ExtraUsage
			extra.DecimalPlaces, extra.UsedCredits, extra.MonthlyLimit, extra.Utilization = &tc.places, &tc.used, &tc.limit, &tc.utilization
			got := usage.ToDisplay()
			if got.Status != StatusConnected || got.Error != "" || len(got.Groups) != 1 || len(got.Groups[0].Buckets) != 1 {
				t.Fatalf("display = %#v", got)
			}
			bucket := got.Groups[0].Buckets[0]
			if bucket.Detail != tc.detail || math.Abs(bucket.Remaining-tc.remaining) > 1e-9 {
				t.Errorf("bucket = %#v", bucket)
			}
		})
	}
}
