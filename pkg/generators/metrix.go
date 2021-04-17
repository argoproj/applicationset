package generators

import (
	"time"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
)

var _ Generator = (*MetrixGenerator)(nil)

type MetrixGenerator struct {
}

func NewMertixGenerator() Generator {
	g := &MetrixGenerator{}
	return g
}

func (g *MetrixGenerator) GenerateParams(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) ([]map[string]string, error) {
	return []map[string]string{}, nil
}

func (g *MetrixGenerator) GetRequeueAfter(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) time.Duration {
	return 0

}

func (g *MetrixGenerator) GetTemplate(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) *argoprojiov1alpha1.ApplicationSetTemplate {
	return nil
}
