package generators

func CombineMaps(a map[string]Generator, b map[string]Generator) map[string]Generator {
	res := map[string]Generator{}

	for k, v := range a {
		res[k] = v
	}

	for k, v := range b {
		res[k] = v
	}

	return res
}
