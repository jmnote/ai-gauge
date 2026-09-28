package providers

import (
	"os"
	"path/filepath"
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
		{"invalid reset", `"reset_after_seconds": 2105558`, `"reset_after_seconds": -1`, false, ""},
		{"zero limit", `"limit": "1000"`, `"limit": "0"`, true, "0/0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			usage := connectedCodex(t, []byte(strings.Replace(fixture, tc.from, tc.to, 1)))
			display := usage.ToDisplay()
			if (display.Error == "") != tc.valid {
				t.Fatalf("error = %q, valid = %v", display.Error, tc.valid)
			}
			if tc.valid && display.Groups[0].Buckets[1].Detail != tc.detail {
				t.Errorf("detail = %q", display.Groups[0].Buckets[1].Detail)
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
