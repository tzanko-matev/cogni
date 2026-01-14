package ratelimiter

// LimitKey identifies the resource being limited.
type LimitKey string

// LimitKind defines the limiter semantics.
type LimitKind string

const (
	// KindRolling enforces a rolling-window capacity.
	KindRolling LimitKind = "rolling"
	// KindConcurrency enforces an in-flight concurrency capacity.
	KindConcurrency LimitKind = "concurrency"
)

// OveragePolicy defines what happens when actual usage exceeds the reservation.
type OveragePolicy string

const (
	// OverageDeny rejects overages once reserved usage is exceeded.
	OverageDeny OveragePolicy = "deny"
	// OverageDebt records overages as debt when reservations fail.
	OverageDebt OveragePolicy = "debt"
)

// LimitStatus tracks whether a limit is active or decreasing.
type LimitStatus string

const (
	// LimitStatusActive allows reservations.
	LimitStatusActive LimitStatus = "active"
	// LimitStatusDecreasing blocks reservations until capacity drops.
	LimitStatusDecreasing LimitStatus = "decreasing"
)

// LimitDefinition is the server-side definition for a limit.
type LimitDefinition struct {
	Key            LimitKey      `json:"key" yaml:"key"`
	Kind           LimitKind     `json:"kind" yaml:"kind"`
	Capacity       uint64        `json:"capacity" yaml:"capacity"`
	WindowSeconds  int           `json:"window_seconds" yaml:"window_seconds"`
	TimeoutSeconds int           `json:"timeout_seconds" yaml:"timeout_seconds"`
	Unit           string        `json:"unit" yaml:"unit"`
	Description    string        `json:"description" yaml:"description"`
	Overage        OveragePolicy `json:"overage" yaml:"overage"`
}

// LimitState captures runtime state for a limit.
type LimitState struct {
	Definition        LimitDefinition `json:"definition" yaml:"definition"`
	Status            LimitStatus     `json:"status" yaml:"status"`
	PendingDecreaseTo uint64          `json:"pending_decrease_to" yaml:"pending_decrease_to"`
}

// Requirement is a requested reservation for a limit.
type Requirement struct {
	Key    LimitKey `json:"key" yaml:"key"`
	Amount uint64   `json:"amount" yaml:"amount"`
}

// Actual reports the actual usage for reconciliation.
type Actual struct {
	Key          LimitKey `json:"key" yaml:"key"`
	ActualAmount uint64   `json:"actual_amount" yaml:"actual_amount"`
}
