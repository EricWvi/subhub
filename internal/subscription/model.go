package subscription

import "time"

type ClashConfigSubscription struct {
	ID          int64                   `json:"id"`
	Name        string                  `json:"name"`
	Providers   []int64                 `json:"providers"`
	ProxyGroups []ClashConfigProxyGroup `json:"proxy_groups"`
	CreatedAt   time.Time               `json:"created_at"`
	UpdatedAt   time.Time               `json:"updated_at"`
}

type ClashConfigProxyGroup struct {
	ID                  int64         `json:"id"`
	Name                string        `json:"name"`
	Type                string        `json:"type"`
	Position            int64         `json:"position"`
	URL                 string        `json:"url"`
	Interval            int64         `json:"interval"`
	Proxies             []ProxyMember `json:"proxies"`
	BindInternalGroupID int64         `json:"bind_internal_proxy_group_id"`
	IsSystem            bool          `json:"is_system"`
}

type ProxyMember struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type ProxyProviderSubscription struct {
	ID                   int64     `json:"id"`
	Name                 string    `json:"name"`
	Providers            []int64   `json:"providers"`
	InternalProxyGroupID int64     `json:"internal_proxy_group_id"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type RuleProviderSubscription struct {
	ID                   int64     `json:"id"`
	Name                 string    `json:"name"`
	Providers            []int64   `json:"providers"`
	InternalProxyGroupID int64     `json:"internal_proxy_group_id"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type CreateClashConfigSubscriptionInput struct {
	Name        string                             `json:"name"`
	Providers   []int64                            `json:"providers"`
	ProxyGroups []CreateClashConfigProxyGroupInput `json:"proxy_groups"`
}

type CreateClashConfigProxyGroupInput struct {
	Name                string        `json:"name"`
	Type                string        `json:"type"`
	Position            int64         `json:"position"`
	URL                 string        `json:"url"`
	Interval            int64         `json:"interval"`
	Proxies             []ProxyMember `json:"proxies"`
	BindInternalGroupID int64         `json:"bind_internal_proxy_group_id"`
}

type UpdateClashConfigSubscriptionInput struct {
	Name        string                             `json:"name"`
	Providers   []int64                            `json:"providers"`
	ProxyGroups []CreateClashConfigProxyGroupInput `json:"proxy_groups"`
}

type CreateProxyProviderSubscriptionInput struct {
	Name                 string  `json:"name"`
	Providers            []int64 `json:"providers"`
	InternalProxyGroupID int64   `json:"internal_proxy_group_id"`
}

type UpdateProxyProviderSubscriptionInput struct {
	Name                 string  `json:"name"`
	Providers            []int64 `json:"providers"`
	InternalProxyGroupID int64   `json:"internal_proxy_group_id"`
}

type CreateRuleProviderSubscriptionInput struct {
	Name                 string  `json:"name"`
	Providers            []int64 `json:"providers"`
	InternalProxyGroupID int64   `json:"internal_proxy_group_id"`
}

type UpdateRuleProviderSubscriptionInput struct {
	Name                 string  `json:"name"`
	Providers            []int64 `json:"providers"`
	InternalProxyGroupID int64   `json:"internal_proxy_group_id"`
}
