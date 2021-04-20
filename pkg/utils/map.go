package utils

func CombineStringMaps(a map[string]string, b map[string]string) map[string]string {
	res := map[string]string{}

	for k, v := range a {
		res[k] = v
	}

	for k, v := range b {
		res[k] = v
	}

	return res
}
