package utils

import (
	"errors"
	"fmt"
)

var DuplicateKey = errors.New("")

func CombineStringMaps(a map[string]string, b map[string]string) (map[string]string, error) {
	res := map[string]string{}

	for k, v := range a {
		res[k] = v
	}

	for k, v := range b {
		current, present := res[k]
		if present && current != v {
			return nil, errors.New(fmt.Sprintf("found duplicate key %s with different value, a: %s ,b: %s", k, current, v))
		}
		res[k] = v
	}

	return res, nil
}
