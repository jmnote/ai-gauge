package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// ParseCodexUsage unmarshals a raw Codex usage response with no validation
// or conversion - see ToDisplay for that.
func ParseCodexUsage(data []byte) (CodexUsage, error) {
	var usage CodexUsage
	if err := json.Unmarshal(data, &usage); err != nil {
		return CodexUsage{}, err
	}
	usage.Raw = append(json.RawMessage(nil), data...)
	return usage, nil
}

// ToDisplay validates the raw rate_limit fields (present, 0-100 for a
// percentage, non-negative for a reset) and converts Codex's relative-
// seconds resets into absolute timestamps anchored to FetchedAt.
func (u CodexUsage) ToDisplay() DisplayUsage {
	email := strings.TrimSpace(u.Email)
	display := DisplayUsage{FetchedAt: u.FetchedAt, User: email, Email: email, ResetCredits: u.RateLimitResetCredits.AvailableCount, DiagnosisFields: u.DiagnosisFields}
	if u.Status != StatusConnected {
		return display
	}

	now := time.Now()
	if parsed, err := time.Parse(time.RFC3339, u.FetchedAt); err == nil {
		now = parsed
	}
	toResetTime := func(seconds int) string {
		if seconds <= 0 {
			return ""
		}
		return now.Add(time.Duration(seconds) * time.Second).Format(time.RFC3339)
	}
	var buckets []DisplayUsageBucket
	for _, window := range []struct {
		name  string
		label string
		data  *CodexUsageWindow
	}{
		{"primary_window", "5h", u.RateLimit.PrimaryWindow},
		{"secondary_window", "7d", u.RateLimit.SecondaryWindow},
	} {
		if window.data == nil {
			continue
		}
		used, reset := window.data.UsedPercent, window.data.ResetAfterSeconds
		var err error
		switch {
		case used == nil:
			err = fmt.Errorf("rate_limit.%s.used_percent is missing or null", window.name)
		case reset == nil:
			err = fmt.Errorf("rate_limit.%s.reset_after_seconds is missing or null", window.name)
		case *used < 0 || *used > 100:
			err = fmt.Errorf("rate_limit.%s.used_percent = %g; expected 0-100", window.name, *used)
		case *reset < 0:
			err = fmt.Errorf("rate_limit.%s.reset_after_seconds = %d; expected >= 0", window.name, *reset)
		}
		if err != nil {
			display.applyDiagnosis(usageUnreadableDiagnosis("Codex", ReasonUnsupportedResponse, err))
			return display
		}
		label := window.label
		if seconds := window.data.LimitWindowSeconds; seconds > 0 {
			switch {
			case seconds%86400 == 0:
				label = fmt.Sprintf("%dd", seconds/86400)
			case seconds%3600 == 0:
				label = fmt.Sprintf("%dh", seconds/3600)
			default:
				label = (time.Duration(seconds) * time.Second).String()
			}
		}
		buckets = append(buckets, DisplayUsageBucket{Label: label, Remaining: 100 - *used, ResetTime: toResetTime(*reset)})
	}

	// Individual limits are optional; malformed values must not hide valid windows.
	if bucket, ok := codexIndividualLimitBucket(u.SpendControl.IndividualLimit); ok {
		bucket.ResetTime = toResetTime(*u.SpendControl.IndividualLimit.ResetAfterSeconds)
		buckets = append(buckets, bucket)
	}
	if len(buckets) == 0 {
		display.applyDiagnosis(usageUnreadableDiagnosis("Codex", ReasonNoUsageData, fmt.Errorf("no usable usage limits available in response")))
		return display
	}

	display.Plan = u.PlanType
	display.Groups = []DisplayUsageGroup{{Buckets: buckets}}
	display.Status = StatusConnected
	return display
}

func codexIndividualLimitBucket(limit *CodexIndividualLimit) (DisplayUsageBucket, bool) {
	if limit == nil || limit.RemainingPercent == nil || limit.ResetAfterSeconds == nil || *limit.ResetAfterSeconds < 0 {
		return DisplayUsageBucket{}, false
	}
	used, usedErr := strconv.ParseFloat(limit.Used, 64)
	total, totalErr := strconv.ParseFloat(limit.Limit, 64)
	if usedErr != nil || totalErr != nil {
		return DisplayUsageBucket{}, false
	}
	for _, value := range []float64{used, total, *limit.RemainingPercent} {
		if value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
			return DisplayUsageBucket{}, false
		}
	}
	if *limit.RemainingPercent > 100 {
		return DisplayUsageBucket{}, false
	}
	detail, amounts := formatUsageDetails(used, math.Max(0, total-used), total)
	return DisplayUsageBucket{Label: "mo", Detail: detail, Amounts: amounts, Remaining: *limit.RemainingPercent}, true
}

func GetCodexUsage(tokenKey string) CodexUsage {
	return getCodexUsage(context.Background(), defaultDeps(), tokenKey, true)
}

// FetchCodexRawUsage returns the unconverted usage response for the Codex
// instance whose token is stored under tokenKey, using the same auth/HTTP
// path as GetCodexUsage. Used by hack/fixtures/fixtures.go to capture the
// API's actual response shape for fixture development.
func FetchCodexRawUsage(tokenKey string) ([]byte, error) {
	ctx := context.Background()
	deps := defaultDeps()
	diagnosis, credentials, ok := diagnoseCodex(ctx, deps, tokenKey, true)
	if !ok {
		return nil, fmt.Errorf("%s", diagnosis.Message)
	}
	accessToken, err := authorizedAccessToken(ctx, deps, "codex", tokenKey, credentials.Tokens.AccessToken)
	if err != nil {
		return nil, err
	}
	return fetchAuthorizedJSON(ctx, "https://chatgpt.com/backend-api/wham/usage", "Codex", map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
}

func getCodexUsage(ctx context.Context, deps providerDeps, tokenKey string, active bool) CodexUsage {
	usage := CodexUsage{FetchedAt: time.Now().Format(time.RFC3339)}

	diagnosis, credentials, ok := diagnoseCodex(ctx, deps, tokenKey, active)
	if !ok {
		usage.applyDiagnosis(diagnosis)
		return usage
	}

	accessToken, err := authorizedAccessToken(ctx, deps, "codex", tokenKey, credentials.Tokens.AccessToken)
	if err != nil {
		usage.applyDiagnosis(usageFailureDiagnosis("Codex", err))
		return usage
	}
	body, err := fetchAuthorizedJSON(ctx, "https://chatgpt.com/backend-api/wham/usage", "Codex", map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if err != nil {
		usage.applyDiagnosis(usageFailureDiagnosis("Codex", err))
		return usage
	}

	parsed, err := ParseCodexUsage(body)
	if err != nil {
		usage.applyDiagnosis(usageUnreadableDiagnosis("Codex", ReasonUnsupportedResponse, err))
		return usage
	}
	parsed.FetchedAt = usage.FetchedAt
	parsed.Status = StatusConnected
	return parsed
}
