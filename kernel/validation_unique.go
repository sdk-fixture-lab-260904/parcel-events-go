package kernel

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// Type tags and exact rationals implement JSON equality without conflating true and 1.
func uniqueKey(value any, depth int, budget *int) (string, error) {
	*budget--
	if depth > 128 || *budget < 0 {
		return "", fmt.Errorf("unique item nesting limit")
	}
	switch v := value.(type) {
	case json.Number:
		n, ok := validationNumber(v)
		if !ok {
			return "", fmt.Errorf("numeric validation limit")
		}
		return "n" + n.RatString(), nil
	case []any:
		parts := make([]string, len(v))
		for i, item := range v {
			key, err := uniqueKey(item, depth+1, budget)
			if err != nil {
				return "", err
			}
			parts[i] = key
		}
		encoded, _ := json.Marshal(parts)
		return "a" + string(encoded), nil
	case map[string]any:
		parts := []string{}
		for _, key := range slices.Sorted(maps.Keys(v)) {
			item, err := uniqueKey(v[key], depth+1, budget)
			if err != nil {
				return "", err
			}
			parts = append(parts, key, item)
		}
		encoded, _ := json.Marshal(parts)
		return "o" + string(encoded), nil
	default:
		encoded, err := json.Marshal(v)
		return "s" + strings.TrimSpace(string(encoded)), err
	}
}
