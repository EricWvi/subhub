package render

import (
	"os"

	"gopkg.in/yaml.v3"
)

func MihomoTemplate(templatePath string, nodes []map[string]any, manualRules []string) (string, error) {
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return "", err
	}
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return "", err
	}

	existing, _ := doc["rules"].([]any)
	merged := make([]any, 0, len(manualRules)+len(existing))
	for _, rule := range manualRules {
		merged = append(merged, rule)
	}
	merged = append(merged, existing...)
	doc["rules"] = merged
	doc["proxies"] = nodes

	out, err := yaml.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
