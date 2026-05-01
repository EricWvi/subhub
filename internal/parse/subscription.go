package parse

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

type ProxySchema struct {
	Proxies []map[string]any `yaml:"proxies"`
}

func DecodeAndNormalize(payload []byte) ([]map[string]any, string, error) {
	trimmed := bytes.TrimSpace(payload)
	if decoded, err := tryBase64(trimmed); err == nil {
		return parseYAML(decoded, "base64+yaml")
	}
	return parseYAML(trimmed, "yaml")
}

func tryBase64(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, base64.CorruptInputError(0)
	}
	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(string(data))
	}
	if err != nil {
		return nil, err
	}
	if len(decoded) == 0 {
		return nil, base64.CorruptInputError(0)
	}
	return decoded, nil
}

func parseYAML(data []byte, format string) ([]map[string]any, string, error) {
	var schema ProxySchema
	if err := yaml.Unmarshal(data, &schema); err != nil {
		return nil, "", fmt.Errorf("unsupported provider payload: %w", err)
	}
	if len(schema.Proxies) == 0 {
		return nil, "", errors.New("unsupported provider payload: proxies list is empty")
	}
	return schema.Proxies, format, nil
}
