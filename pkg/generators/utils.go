package generators

import (
	"fmt"
)

func CombineMaps(a map[string]Generator, b map[string]Generator) (map[string]Generator, error) {
	res := map[string]Generator{}

	for k, v := range a {
		res[k] = v
	}

	for k, v := range b {
		current, present := res[k]
		if present && current != v {
			return nil, fmt.Errorf("found duplicate key %s with different value, a: %s ,b: %s", k, current, v)
		}
		res[k] = v
	}

	return res, nil
}
