package generators

import (
	"errors"
	"time"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/argoproj-labs/applicationset/pkg/utils"
)

var _ Generator = (*MatrixGenerator)(nil)

var MoreThanTwoGenerators = errors.New("found more than two generators, Matrix support only two")
var LessThanTwoGenerators = errors.New("found less than two generators, Matrix support only two")
var MoreThenOneInnerGenerators = errors.New("found more than one generator in matrix.Generators")

type MatrixGenerator struct {
	// The inner generators supported by the matrix generator (cluster, git, list...)
	supportedGenerators map[string]Generator
}

func NewMatrixGenerator(supportedGenerators map[string]Generator) Generator {
	m := &MatrixGenerator{
		supportedGenerators: supportedGenerators,
	}
	return m
}

func (m *MatrixGenerator) GenerateParams(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator, appSet *argoprojiov1alpha1.ApplicationSet) ([]map[string]string, error) {

	if len(appSetGenerator.Matrix.Generators) < 2 {
		return nil, LessThanTwoGenerators
	}

	if len(appSetGenerator.Matrix.Generators) > 2 {
		return nil, MoreThanTwoGenerators
	}

	res := []map[string]string{}

	g0, err := m.getParams(appSetGenerator.Matrix.Generators[0], appSet)
	if err != nil {
		return nil, err
	}
	g1, err := m.getParams(appSetGenerator.Matrix.Generators[1], appSet)
	if err != nil {
		return nil, err
	}

	for _, a := range g0 {
		for _, b := range g1 {
			val, err := utils.CombineStringMaps(a, b)
			if err != nil {
				return nil, err
			}
			res = append(res, val)
		}
	}

	return res, nil
}

func (m *MatrixGenerator) getParams(appSetBaseGenerator argoprojiov1alpha1.ApplicationSetBaseGenerator, appSet *argoprojiov1alpha1.ApplicationSet) ([]map[string]string, error) {

	t, _ := Transform(
		argoprojiov1alpha1.ApplicationSetGenerator{
			List:     appSetBaseGenerator.List,
			Clusters: appSetBaseGenerator.Clusters,
			Git:      appSetBaseGenerator.Git,
		},
		m.supportedGenerators,
		argoprojiov1alpha1.ApplicationSetTemplate{},
		appSet)

	if len(t) > 1 {
		return nil, MoreThenOneInnerGenerators
	}

	return t[0].Params, nil
}

const maxDuration time.Duration = 1<<63 - 1

func (m *MatrixGenerator) GetRequeueAfter(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) time.Duration {
	res := maxDuration
	var found bool

	for _, r := range appSetGenerator.Matrix.Generators {
		base := &argoprojiov1alpha1.ApplicationSetGenerator{
			List:     r.List,
			Clusters: r.Clusters,
			Git:      r.Git,
		}
		generators := GetRelevantGenerators(base, m.supportedGenerators)

		for _, g := range generators {
			temp := g.GetRequeueAfter(base)
			if temp < res && temp != NoRequeueAfter {
				found = true
				res = temp
			}
		}
	}

	if found {
		return res
	} else {
		return NoRequeueAfter
	}

}

func (m *MatrixGenerator) GetTemplate(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) *argoprojiov1alpha1.ApplicationSetTemplate {
	return &appSetGenerator.Matrix.Template
}
