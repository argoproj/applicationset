package generators

import (
	"reflect"
	"time"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	log "github.com/sirupsen/logrus"
)

var _ Generator = (*MetrixGenerator)(nil)

type MetrixGenerator struct {
	generators map[string]Generator
}

func NewMertixGenerator(generators map[string]Generator) Generator {
	g := &MetrixGenerator{
		generators: generators,
	}
	return g
}

func (m *MetrixGenerator) GetRelevantGenerators(requestedGenerator *argoprojiov1alpha1.ApplicationSetGenerator) []Generator {
	var res []Generator

	v := reflect.Indirect(reflect.ValueOf(requestedGenerator))
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.CanInterface() {
			continue
		}

		if !reflect.ValueOf(field.Interface()).IsNil() {
			res = append(res, m.generators[v.Type().Field(i).Name])
		}
	}

	return res
}

func combineMaps(a map[string]string, b map[string]string) map[string]string {
	res := map[string]string{}

	for k, v := range a {
		res[k] = v
	}

	for k, v := range b {
		res[k] = v
	}

	return res
}

func (m *MetrixGenerator) GenerateParams(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) ([]map[string]string, error) {

	allParams := [][]map[string]string{}

	for _, requestedGenerator := range appSetGenerator.Metrix.Generators {
		generator := m.GetRelevantGenerators(&requestedGenerator)[0]

		params, err := generator.GenerateParams(&requestedGenerator)
		if err != nil {
			log.WithError(err).WithField("generator", generator).
				Error("error generating params")
			//if firstError == nil {
			//	firstError = err
			//}
			continue
		}

		allParams = append(allParams, params)
	}

	res := []map[string]string{}

	for _, a := range allParams[0] {
		for _, b := range allParams[1] {
			res = append(res, combineMaps(a, b))
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
