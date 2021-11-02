package generators

import (
	"errors"
	"fmt"
	"time"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/argoproj-labs/applicationset/pkg/utils"
)

var _ Generator = (*UnionGenerator)(nil)

var LessThanTwoGeneratorsInUnion = errors.New("found less than two generators, Union requires two or more")

type UnionGenerator struct {
	// The inner generators supported by the union generator (cluster, git, list...)
	supportedGenerators map[string]Generator
}

func NewUnionGenerator(supportedGenerators map[string]Generator) Generator {
	m := &UnionGenerator{
		supportedGenerators: supportedGenerators,
	}
	return m
}

// keysArePresentAndValuesAreEqual returns true if each key is present in both maps and the respective values for each key are equal between the two maps
func keysArePresentAndValuesAreEqual(keys []string, a map[string]string, b map[string]string) bool {
	for _, key := range keys {
		aValue, aPresent := a[key]
		if !aPresent {
			return false
		}
		bValue, bPresent := b[key]
		if !bPresent {
			return false
		}
		if aValue != bValue {
			return false
		}
	}
	return true
}

func (m *UnionGenerator) getParamsForAllGenerators(generators []argoprojiov1alpha1.ApplicationSetBaseGenerator, appSet *argoprojiov1alpha1.ApplicationSet) ([]map[string]string, error) {
	var paramSets []map[string]string
	for _, generator := range generators {
		generatorParamSets, err := m.getParams(generator, appSet)
		if err != nil {
			return nil, err
		}
		// concatenate param lists produced by each generator
		for _, generatorParams := range generatorParamSets {
			paramSets = append(paramSets, generatorParams)
		}
	}
	return paramSets, nil
}

// tryMergeParamSets merges `a` and `b` if and only if all merge keys are present in both maps and their respective values are equal.
// If the maps aren't merged, returns `a` unchanged.
func tryMergeParamSets(mergeKeys []string, a map[string]string, b map[string]string) (params map[string]string, wasMerged bool, err error) {
	if !keysArePresentAndValuesAreEqual(mergeKeys, a, b) {
		return a, false, nil
	}
	merged, err := utils.CombineStringMapsAllowDuplicates(a, b)
	if err != nil {
		return a, false, err
	}
	return merged, true, nil
}

func (m *UnionGenerator) GenerateParams(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator, appSet *argoprojiov1alpha1.ApplicationSet) ([]map[string]string, error) {
	if len(appSetGenerator.Union.Generators) < 2 {
		return nil, LessThanTwoGeneratorsInUnion
	}

	var paramSets []map[string]string

	paramSetsFromGenerators, err := m.getParamsForAllGenerators(appSetGenerator.Union.Generators, appSet)
	if err != nil {
		return nil, err
	}

	paramSetAlreadyHandled := make([]bool, len(paramSetsFromGenerators))

	// merge any param sets which have matching merge keys
	for i, paramsFromGenerator := range paramSetsFromGenerators {
		if paramSetAlreadyHandled[i] {
			continue
		}

		var paramsToUse = paramsFromGenerator

		// look in remaining param sets for a set which can be merged
		for j, paramsToMaybeMerge := range paramSetsFromGenerators[i+1:] {
			if paramSetAlreadyHandled[j] {
				continue
			}
			var wasMerged bool
			paramsToUse, wasMerged, err = tryMergeParamSets(appSetGenerator.Union.MergeKeys, paramsToUse, paramsToMaybeMerge)
			if err != nil {
				return nil, err
			}
			if wasMerged {
				paramSetAlreadyHandled[j+1] = true
			}
		}
		paramSets = append(paramSets, paramsToUse)
	}

	return paramSets, nil
}

func (m *UnionGenerator) getParams(appSetBaseGenerator argoprojiov1alpha1.ApplicationSetBaseGenerator, appSet *argoprojiov1alpha1.ApplicationSet) ([]map[string]string, error) {

	t, err := Transform(
		argoprojiov1alpha1.ApplicationSetGenerator{
			List:                    appSetBaseGenerator.List,
			Clusters:                appSetBaseGenerator.Clusters,
			Git:                     appSetBaseGenerator.Git,
			SCMProvider:             appSetBaseGenerator.SCMProvider,
			ClusterDecisionResource: appSetBaseGenerator.ClusterDecisionResource,
			PullRequest:             appSetBaseGenerator.PullRequest,
		},
		m.supportedGenerators,
		argoprojiov1alpha1.ApplicationSetTemplate{},
		appSet)

	if err != nil {
		return nil, fmt.Errorf("child generator returned an error on parameter generation: %v", err)
	}

	if len(t) == 0 {
		return nil, fmt.Errorf("child generator generated no parameters")
	}

	if len(t) > 1 {
		return nil, MoreThenOneInnerGenerators
	}

	return t[0].Params, nil
}

func (m *UnionGenerator) GetRequeueAfter(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) time.Duration {
	res := maxDuration
	var found bool

	for _, r := range appSetGenerator.Union.Generators {
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

func (m *UnionGenerator) GetTemplate(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) *argoprojiov1alpha1.ApplicationSetTemplate {
	return &appSetGenerator.Union.Template
}
