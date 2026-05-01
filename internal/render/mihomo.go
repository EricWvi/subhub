package render

import (
	"os"

	"gopkg.in/yaml.v3"
)

func MihomoTemplate(templatePath string, nodes []map[string]any) (string, error) {
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return "", err
	}
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return "", err
	}
	doc["proxies"] = nodes
	out, err := yaml.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
