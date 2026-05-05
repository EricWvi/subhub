package rule

import "time"

type Rule struct {
	ID         int64     `json:"id"`
	RuleType   string    `json:"rule_type"`
	Pattern    string    `json:"pattern"`
	ProxyGroup string    `json:"proxy_group"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CreateRuleInput struct {
	RuleType   string `json:"rule_type"`
	Pattern    string `json:"pattern"`
	ProxyGroup string `json:"proxy_group"`
}

type UpdateRuleInput struct {
	RuleType   string `json:"rule_type"`
	Pattern    string `json:"pattern"`
	ProxyGroup string `json:"proxy_group"`
}

type ImportRulesInput struct {
	Rules   string `json:"rules"`
	Reverse bool   `json:"reverse"`
}

type ImportRulesResult struct {
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
}

type ListRulesResult struct {
	Rules    []Rule `json:"rules"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Total    int    `json:"total"`
}
