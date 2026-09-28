package providers

import "encoding/json"

// ClaudeUsage mirrors https://api.anthropic.com/api/oauth/usage's response
// field-for-field - this is what a raw hack/fixtures/usage/usage_claude_*.json
// fixture looks like. Plan and FetchedAt aren't part of that response: Plan
// comes from the credentials file's subscriptionType and FetchedAt is
// stamped on at fetch time, both attached after the fact. ToDisplay
// (claude.go) does all validation and unit conversion; this type does none.
type ClaudeUsage struct {
	FiveHour           claudeUsageWindow  `json:"five_hour"`
	SevenDay           claudeUsageWindow  `json:"seven_day"`
	SevenDayOpus       *claudeUsageWindow `json:"seven_day_opus"`
	SevenDaySonnet     *claudeUsageWindow `json:"seven_day_sonnet"`
	ExtraUsage         *claudeExtraUsage  `json:"extra_usage"`
	Plan               string             `json:"plan"`
	FetchedAt          string             `json:"fetchedAt"`
	AccountDisplayName string             `json:"-"`

	// Raw is the complete, untrimmed response body ParseClaudeUsage was
	// given - this endpoint is undocumented and returns several fields
	// (feature-flag-looking names, a spend object) this type
	// doesn't name yet, so nothing is silently dropped for a future
	// ToDisplay to draw on.
	Raw json.RawMessage `json:"-"`

	DiagnosisFields
}

type claudeUsageWindow struct {
	Utilization *float64 `json:"utilization"`
	ResetsAt    string   `json:"resets_at"`
}

type claudeExtraUsage struct {
	IsEnabled     bool     `json:"is_enabled"`
	MonthlyLimit  *float64 `json:"monthly_limit"`
	UsedCredits   *float64 `json:"used_credits"`
	Utilization   *float64 `json:"utilization"`
	DecimalPlaces *int     `json:"decimal_places"`
}

// claudeCredentials is the shape of the Claude CLI's own credentials file
// (~/.claude/.credentials.json), not the usage API - unrelated to
// ClaudeUsage above.
type claudeCredentials struct {
	ClaudeAiOauth struct {
		AccessToken      string `json:"accessToken"`
		SubscriptionType string `json:"subscriptionType"`
	} `json:"claudeAiOauth"`
	AccountDisplayName string `json:"accountDisplayName"`
}

type claudeProfile struct {
	Account struct {
		DisplayName string `json:"display_name"`
	} `json:"account"`
}
