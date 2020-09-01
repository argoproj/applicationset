package generators

import (
	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
)

// Generator defines the interface implemented by all ApplicationSet generators.
type Generator interface {
	// GenerateApplications interprets the ApplicationSet and generates all relevant Applications.
	// The expected / desired list of Applications is returned, it then needs to be reconciled
	// against the current state of the Applications in the cluster.
	GenerateParams(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) ([]map[string]string, error)
}

type EmptyAppSetGeneratorError struct {

}

func (e EmptyAppSetGeneratorError) Error() string{
	return "ApplicationSet is empty"
}

