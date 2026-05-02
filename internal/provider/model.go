package provider

import "time"

type Provider struct {
	ID                     int64     `json:"id"`
	Name                   string    `json:"name"`
	URL                    string    `json:"url"`
	RefreshIntervalMinutes int64     `json:"refresh_interval_minutes"`
	Abbrev                 string    `json:"abbrev"`
	Used                   int64     `json:"used"`
	Total                  int64     `json:"total"`
	Expire                 int64     `json:"expire"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
	LastRefreshStatus      string    `json:"last_refresh_status,omitempty"`
	LastRefreshMessage     string    `json:"last_refresh_message,omitempty"`
}

type ProxyNode struct {
	ID         int64
	ProviderID int64
	Name       string
	RawYAML    string
	UpdateMark int64
}

type Snapshot struct {
	ID              int64     `json:"id"`
	ProviderID      int64     `json:"provider_id"`
	Format          string    `json:"format"`
	NormalizedYAML  string    `json:"normalized_yaml"`
	NodeCount       int       `json:"node_count"`
	FetchedAt       time.Time `json:"fetched_at"`
}

type RefreshAttempt struct {
	ID          int64
	ProviderID  int64
	Status      string
	Message     string
	AttemptedAt time.Time
}

type RefreshFailedError struct {
	Message string
}

func (e *RefreshFailedError) Error() string {
	return e.Message
}
