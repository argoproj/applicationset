package generators

import (
	"errors"
	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
)

// Generator defines the interface implemented by all ApplicationSet generators.
type Generator interface {
	// GenerateParams interprets the ApplicationSet and generates all relevant parameters for the application template.
	// This function will be called for every generator,
	//even if there isn't any application - in this case the result should be nil without an error
	// The expected / desired list of parameters is returned, it then needs to be render and reconciled
	// against the current state of the Applications in the cluster.
	GenerateParams(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) ([]map[string]string, error)
}

var EmptyAppSetGeneratorError = errors.New("ApplicationSet is empty")