package provider

import "time"

type Provider struct {
	ID                    int64
	Name                  string
	URL                   string
	RefreshIntervalSeconds int
	CreatedAt             time.Time
	UpdatedAt             time.Time
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
