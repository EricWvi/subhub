package render

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type RenderedProxyGroup struct {
	Name     string
	Type     string
	URL      string
	Interval int64
	Proxies  []string
}

func RenderClashConfigSubscription(templatePath string, proxies []map[string]any, groups []RenderedProxyGroup, rules []string) (string, error) {
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return "", err
	}
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return "", err
	}

	doc["proxies"] = proxies

	existingGroups, _ := doc["proxy-groups"].([]any)

	subscriptionNames := map[string]bool{}
	var proxyGroups []map[string]any
	for _, g := range groups {
		pg := map[string]any{
			"name": g.Name,
			"type": g.Type,
		}
		if g.URL != "" {
			pg["url"] = g.URL
		}
		if g.Interval > 0 {
			pg["interval"] = g.Interval
		}
		proxyMembers := make([]any, len(g.Proxies))
		for i, p := range g.Proxies {
			proxyMembers[i] = p
		}
		pg["proxies"] = proxyMembers
		proxyGroups = append(proxyGroups, pg)
		subscriptionNames[g.Name] = true
	}
	for _, eg := range existingGroups {
		if m, ok := eg.(map[string]any); ok {
			if n, ok := m["name"].(string); ok && !subscriptionNames[n] {
				proxyGroups = append(proxyGroups, m)
			}
		}
	}
	doc["proxy-groups"] = proxyGroups

	existingRules, _ := doc["rules"].([]any)
	merged := make([]any, 0, len(rules)+len(existingRules))
	for _, rule := range rules {
		merged = append(merged, rule)
	}
	merged = append(merged, existingRules...)
	doc["rules"] = merged

	out, err := yaml.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func RenderProxyProviderSubscription(nodes []map[string]any) (string, error) {
	out, err := yaml.Marshal(map[string]any{"proxies": nodes})
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func RenderRuleProviderSubscription(rules []string) (string, error) {
	payload := make([]any, 0, len(rules))
	for _, rule := range rules {
		parts := strings.SplitN(rule, ",", 3)
		if len(parts) >= 2 {
			payload = append(payload, parts[0]+","+parts[1])
		} else {
			payload = append(payload, rule)
		}
	}
	out, err := yaml.Marshal(map[string]any{"payload": payload})
	if err != nil {
		return "", err
	}
	return string(out), nil
}
