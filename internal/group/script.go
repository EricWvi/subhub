package group

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/dop251/goja"
)

func SelectNodeIDs(script string, nodes []ProxyNodeView) ([]int64, error) {
	if strings.TrimSpace(script) == "" {
		return allNodeIDs(nodes), nil
	}

	vm := goja.New()
	value, err := vm.RunString("(" + script + ")")
	if err != nil {
		return nil, err
	}

	fn, ok := goja.AssertFunction(value)
	if !ok {
		return nil, errors.New("script must evaluate to a function")
	}

	arg, err := toJSONObject(nodes)
	if err != nil {
		return nil, err
	}
	result, err := fn(goja.Undefined(), vm.ToValue(arg))
	if err != nil {
		return nil, err
	}

	return exportNodeIDs(result.Export(), nodes)
}

func toJSONObject(v any) (any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func allNodeIDs(nodes []ProxyNodeView) []int64 {
	ids := make([]int64, len(nodes))
	for i, node := range nodes {
		ids[i] = node.ID
	}
	return ids
}

func exportNodeIDs(value any, nodes []ProxyNodeView) ([]int64, error) {
	raw, ok := value.([]any)
	if !ok {
		return nil, errors.New("script must return number[]")
	}

	allowed := map[int64]struct{}{}
	for _, node := range nodes {
		allowed[node.ID] = struct{}{}
	}

	var ids []int64
	seen := map[int64]struct{}{}
	for _, item := range raw {
		number, ok := item.(int64)
		if !ok {
			floatNumber, floatOK := item.(float64)
			if !floatOK {
				return nil, errors.New("script must return number[]")
			}
			number = int64(floatNumber)
			if float64(number) != floatNumber {
				return nil, errors.New("script must return whole-number ids")
			}
		}
		if _, ok := allowed[number]; !ok {
			return nil, errors.New("script returned unknown proxy node id")
		}
		if _, ok := seen[number]; ok {
			return nil, errors.New("script returned duplicate proxy node id")
		}
		seen[number] = struct{}{}
		ids = append(ids, number)
	}
	return ids, nil
}
