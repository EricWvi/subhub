package provider

import "time"

type Provider struct {
	ID                     int64     `json:"id"`
	Name                   string    `json:"name"`
	URL                    string    `json:"url"`
	RefreshIntervalSeconds int64     `json:"refresh_interval_seconds"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type Snapshot struct {
	ID             int64
	ProviderID     int64
	Format         string
	RawPayload     []byte
	NormalizedYAML string
	NodeCount      int
	FetchedAt      time.Time
	IsLastKnownGood bool
}

type RefreshAttempt struct {
	ID          int64
	ProviderID  int64
	Status      string
	Message     string
	AttemptedAt time.Time
}
