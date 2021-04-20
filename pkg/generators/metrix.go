package generators

import (
	"time"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/argoproj-labs/applicationset/pkg/utils"
)

var _ Generator = (*MetrixGenerator)(nil)

type MetrixGenerator struct {
	generators map[string]Generator
}

func NewMertixGenerator(generators map[string]Generator) Generator {
	m := &MetrixGenerator{
		generators: generators,
	}
	return m
}

func (m *MetrixGenerator) GenerateParams(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) ([]map[string]string, error) {

	// Log a warning if there are unrecognized generators
	//utils.CheckInvalidGenerators(&applicationSetInfo)

	allParams := [][]map[string]string{}

	for _, requestedGenerator := range appSetGenerator.Metrix.Generators {

		t, _ := Transform(requestedGenerator, m.generators, argoprojiov1alpha1.ApplicationSetTemplate{})

		allParams = append(allParams, t[0].Params)
	}

	res := []map[string]string{}

	for _, a := range allParams[0] {
		for _, b := range allParams[1] {
			res = append(res, utils.CombineStringMaps(a, b))
		}
	}

	return res, nil
}

func (g *MetrixGenerator) GetRequeueAfter(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) time.Duration {
	return 0

}

func (g *MetrixGenerator) GetTemplate(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) *argoprojiov1alpha1.ApplicationSetTemplate {
	return nil
}