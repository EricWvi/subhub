package subscription

import "time"

type NodeProviderSubscription struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Nodes     []int64   `json:"nodes"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateNodeProviderSubscriptionInput struct {
	Name  string  `json:"name"`
	Nodes []int64 `json:"nodes"`
}

type UpdateNodeProviderSubscriptionInput = CreateNodeProviderSubscriptionInput
