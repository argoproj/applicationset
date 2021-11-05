package generators

import (
	"testing"

	"github.com/argoproj-labs/applicationset/api/v1alpha1"
)

func TestGetRelevantGenerators(t *testing.T) {
	requestedGenerator := &v1alpha1.ApplicationSetGenerator{
		ApplicationSetTerminalGenerator: &v1alpha1.ApplicationSetTerminalGenerator{
			List: &v1alpha1.ListGenerator{},
		},
	}
	allGenerators := map[string]Generator{
		"List": NewListGenerator(),
	}
	relevantGenerators := GetRelevantGenerators(requestedGenerator, allGenerators)

	for _, generator := range relevantGenerators {
		if generator == nil {
			t.Fatal(`GetRelevantGenerators produced a nil generator`)
		}
	}

	numRelevantGenerators := len(relevantGenerators)
	if numRelevantGenerators != 1 {
		t.Fatalf(`GetRelevantGenerators produced %d generators instead of the expected 1`, numRelevantGenerators)
	}
}
