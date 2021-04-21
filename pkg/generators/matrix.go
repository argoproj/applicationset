package generators

import (
	"time"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/argoproj-labs/applicationset/pkg/utils"
)

var _ Generator = (*MatrixGenerator)(nil)

type MatrixGenerator struct {
	generators map[string]Generator
}

func NewMatrixGenerator(generators map[string]Generator) Generator {
	m := &MatrixGenerator{
		generators: generators,
	}
	return m
}

func (m *MatrixGenerator) GenerateParams(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) ([]map[string]string, error) {

	// Log a warning if there are unrecognized generators
	//utils.CheckInvalidGenerators(&applicationSetInfo)

	allParams := [][]map[string]string{}

	for _, requestedGenerator := range appSetGenerator.Matrix.Generators {

		f := argoprojiov1alpha1.ApplicationSetGenerator{
			List:     requestedGenerator.List,
			Clusters: requestedGenerator.Clusters,
			Git:      requestedGenerator.Git,
		}

		t, _ := Transform(f, m.generators, argoprojiov1alpha1.ApplicationSetTemplate{})

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

func (g *MatrixGenerator) GetRequeueAfter(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) time.Duration {
	return NoRequeueAfter
}

func (g *MatrixGenerator) GetTemplate(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) *argoprojiov1alpha1.ApplicationSetTemplate {
	return &appSetGenerator.Matrix.Template
}
