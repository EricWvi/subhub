package group

import "time"

type ProxyGroup struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Script    string    `json:"script"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ProxyNodeView struct {
	ID           int64  `json:"id"`
	ProviderName string `json:"providerName"`
	Name         string `json:"name"`
}

type ResolvedNode struct {
	ID           int64  `json:"id"`
	ProviderID   int64  `json:"provider_id"`
	ProviderName string `json:"providerName"`
	Name         string `json:"name"`
	RawYAML      string `json:"raw_yaml"`
}
