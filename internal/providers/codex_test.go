package providers

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func capturedUsageFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "hack", "fixtures", "usage", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestCodexCapturedUsage(t *testing.T) {
	for _, tc := range []struct {
		file, label, detail string
		remaining           float64
	}{
		{"usage_codex.json", "5h", "", 0},
		{"usage_codex.monthly.json", "7d", "963/1000", 96},
	} {
		t.Run(tc.file, func(t *testing.T) {
			usage := connectedCodex(t, capturedUsageFixture(t, tc.file))
			usage.FetchedAt = "2026-08-01T00:00:00Z"
			display := usage.ToDisplay()
			if display.Status != StatusConnected || display.Error != "" {
				t.Fatalf("display = %#v", display)
			}
			if len(display.Groups) != 1 || len(display.Groups[0].Buckets) != 2 {
				t.Fatalf("groups = %#v", display.Groups)
			}
			buckets := display.Groups[0].Buckets
			if buckets[0].Label != tc.label {
				t.Errorf("first label = %q", buckets[0].Label)
			}
			if buckets[1].Detail != tc.detail {
				t.Errorf("detail = %q, want %q", buckets[1].Detail, tc.detail)
			}
			if tc.detail != "" {
				if buckets[1].Amounts == nil || *buckets[1].Amounts != (DisplayUsageAmounts{Used: "36.8", Limit: "1000", Remaining: "963.2"}) {
					t.Errorf("amounts = %#v, want separate full values", buckets[1].Amounts)
				}
				wantReset := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC).Add(2105558 * time.Second).Format(time.RFC3339)
				if buckets[1].Label != "mo" || buckets[1].Remaining != tc.remaining || buckets[1].ResetTime != wantReset {
					t.Errorf("individual limit = %#v", buckets[1])
				}
			} else if buckets[1].Label != "7d" {
				t.Errorf("second label = %q", buckets[1].Label)
			}
		})
	}
}

func TestCodexIndividualLimitValidation(t *testing.T) {
	fixture := string(capturedUsageFixture(t, "usage_codex.monthly.json"))
	for _, tc := range []struct {
		name, from, to string
		valid          bool
		detail         string
	}{
		{"valid", `"individual_limit": {`, `"individual_limit": {`, true, "963/1000"},
		{"invalid amount", `"used": "36.79748725891113"`, `"used": "NaN"`, false, ""},
		{"negative limit", `"limit": "1000"`, `"limit": "-1"`, false, ""},
		{"missing amount", `"used": "36.79748725891113"`, `"used": null`, false, ""},
		{"invalid percentage", `"remaining_percent": 96`, `"remaining_percent": 101`, false, ""},
		{"null reset", `"reset_after_seconds": 2105558`, `"reset_after_seconds": null`, false, ""},
		{"null percentage", `"remaining_percent": 96`, `"remaining_percent": null`, false, ""},
		{"missing limit", `"limit": "1000"`, `"limit": null`, false, ""},
		{"infinite amount", `"used": "36.79748725891113"`, `"used": "Inf"`, false, ""},
		{"invalid reset", `"reset_after_seconds": 2105558`, `"reset_after_seconds": -1`, false, ""},
		{"zero limit", `"limit": "1000"`, `"limit": "0"`, true, "0/0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			usage := connectedCodex(t, []byte(strings.Replace(fixture, tc.from, tc.to, 1)))
			usage.FetchedAt = "2026-08-01T00:00:00Z"
			baseline := usage
			baseline.SpendControl.IndividualLimit = nil
			expected := baseline.ToDisplay()
			display := usage.ToDisplay()
			if display.Status != StatusConnected || display.Error != "" || len(display.Groups) != 1 {
				t.Fatalf("optional limit must not hide valid windows: %#v", display)
			}
			if tc.valid {
				if len(display.Groups[0].Buckets) != 2 || display.Groups[0].Buckets[1].Detail != tc.detail {
					t.Fatalf("buckets = %#v, want valid monthly limit", display.Groups[0].Buckets)
				}
				if tc.name == "zero limit" && display.Groups[0].Buckets[1].Remaining != 0 {
					t.Errorf("zero limit remaining = %g, want 0", display.Groups[0].Buckets[1].Remaining)
				}
			} else if !reflect.DeepEqual(display.Groups, expected.Groups) {
				t.Fatalf("invalid optional limit changed valid windows: %#v", display.Groups)
			}
			// Also preserve both normal 5h/7d windows, including reset timestamps.
			regular := connectedCodex(t, readFixture(t, "codex-usage.json"))
			regular.FetchedAt = usage.FetchedAt
			before := regular.ToDisplay()
			regular.SpendControl.IndividualLimit = usage.SpendControl.IndividualLimit
			if !tc.valid {
				after := regular.ToDisplay()
				if after.Status != StatusConnected || after.Error != "" || !reflect.DeepEqual(after.Groups, before.Groups) {
					t.Fatalf("invalid optional limit changed 5h/7d windows: %#v", after)
				}
			}
			usage.RateLimit.PrimaryWindow = nil
			usage.RateLimit.SecondaryWindow = nil
			monthlyOnly := usage.ToDisplay()
			if tc.valid {
				if monthlyOnly.Status != StatusConnected || monthlyOnly.Error != "" || len(monthlyOnly.Groups) != 1 || len(monthlyOnly.Groups[0].Buckets) != 1 {
					t.Fatalf("valid monthly-only response = %#v", monthlyOnly)
				}
			} else if monthlyOnly.Error == "" || monthlyOnly.Reason != ReasonNoUsageData || len(monthlyOnly.Groups) != 0 {
				t.Fatalf("invalid monthly-only response = %#v, want no usage data", monthlyOnly)
			}
		})
	}
	usage := connectedCodex(t, []byte(fixture))
	usage.SpendControl.IndividualLimit = nil
	if display := usage.ToDisplay(); display.Error != "" || len(display.Groups[0].Buckets) != 1 {
		t.Fatalf("null individual limit = %#v", display)
	}
	usage = connectedCodex(t, []byte(fixture))
	usage.RateLimit.PrimaryWindow = nil
	if display := usage.ToDisplay(); display.Error != "" || len(display.Groups[0].Buckets) != 1 {
		t.Fatalf("individual limit only = %#v", display)
	}
}

func TestCodexWindowDurationLabels(t *testing.T) {
	for _, tc := range []struct {
		seconds int
		label   string
	}{
		{1800, "30m"},
		{60, "1m"},
		{5400, "90m"},
		{18000, "5h"},
		{604800, "7d"},
		{90, "1m30s"},
	} {
		t.Run(tc.label, func(t *testing.T) {
			usage := connectedCodex(t, readFixture(t, "codex-usage.json"))
			usage.RateLimit.PrimaryWindow.LimitWindowSeconds = tc.seconds
			display := usage.ToDisplay()
			if display.Error != "" || len(display.Groups) != 1 || len(display.Groups[0].Buckets) != 2 {
				t.Fatalf("display = %#v", display)
			}
			if got := display.Groups[0].Buckets[0].Label; got != tc.label {
				t.Errorf("label for %d seconds = %q, want %q", tc.seconds, got, tc.label)
			}
		})
	}
}
