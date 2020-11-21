package generators

import (
	"errors"
	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"time"
)

// Generator defines the interface implemented by all ApplicationSet generators.
type Generator interface {
	// GenerateParams interprets the ApplicationSet and generates all relevant parameters for the application template.
	// The expected / desired list of parameters is returned, it then will be render and reconciled
	// against the current state of the Applications in the cluster.
	GenerateParams(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) ([]map[string]string, error)

	// GetRequeueAfter is the the generator can controller the next reconciled loop
	// In case there is more then one generator the time will be the minimum of the times.
	// In case NoRequeueAfter is empty, it will be ignored
	GetRequeueAfter(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) time.Duration
}

var EmptyAppSetGeneratorError = errors.New("ApplicationSet is empty")
var NoRequeueAfter time.Duration
