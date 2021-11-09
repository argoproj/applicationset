package generators

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/argoproj-labs/applicationset/pkg/utils"
)

var _ Generator = (*MergeGenerator)(nil)

var LessThanTwoGeneratorsInMerge = errors.New("found less than two generators, Merge requires two or more")
var NoMergeKeys = errors.New("no merge keys were specified, Merge requires at least one")
var NonUniqueParamSets = errors.New("the parameters from a generator were not unique by the given mergeKeys, Merge requires all param sets to be unique")

type MergeGenerator struct {
	// The inner generators supported by the merge generator (cluster, git, list...)
	supportedGenerators map[string]Generator
}

func NewMergeGenerator(supportedGenerators map[string]Generator) Generator {
	m := &MergeGenerator{
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

func (m *MergeGenerator) getParamSetsForAllGenerators(generators []argoprojiov1alpha1.ApplicationSetNestedGenerator, appSet *argoprojiov1alpha1.ApplicationSet) ([][]map[string]string, error) {
	var paramSets [][]map[string]string
	for _, generator := range generators {
		generatorParamSets, err := m.getParams(generator, appSet)
		if err != nil {
			return nil, err
		}
		// concatenate param lists produced by each generator
		paramSets = append(paramSets, generatorParamSets)
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

func (m *MergeGenerator) GenerateParams(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator, appSet *argoprojiov1alpha1.ApplicationSet) ([]map[string]string, error) {
	if appSetGenerator.Merge == nil {
		return nil, nil
	}

	if len(appSetGenerator.Merge.Generators) < 2 {
		return nil, LessThanTwoGeneratorsInMerge
	}

	paramSetsFromGenerators, err := m.getParamSetsForAllGenerators(appSetGenerator.Merge.Generators, appSet)
	if err != nil {
		return nil, err
	}

	baseParamSetsByMergeKey, err := getParamSetsByMergeKey(appSetGenerator.Merge.MergeKeys, paramSetsFromGenerators[0])

	for _, paramSets := range paramSetsFromGenerators[1:] {
		paramSetsByMergeKey, err := getParamSetsByMergeKey(appSetGenerator.Merge.MergeKeys, paramSets)
		if err != nil {
			return nil, err
		}

		for mergeKeyValue, baseParamSet := range baseParamSetsByMergeKey {
			if overrideParamSet, exists := paramSetsByMergeKey[mergeKeyValue]; exists {
				overriddenParamSet, err := utils.CombineStringMapsAllowDuplicates(baseParamSet, overrideParamSet)
				if err != nil {
					return nil, err
				}
				baseParamSetsByMergeKey[mergeKeyValue] = overriddenParamSet
			}
		}
	}

	mergedParamSets := make([]map[string]string, len(baseParamSetsByMergeKey))
	var i = 0
	for _, mergedParamSet := range baseParamSetsByMergeKey {
		mergedParamSets[i] = mergedParamSet
		i += 1
	}

	return mergedParamSets, nil
}

func getParamSetsByMergeKey(mergeKeys []string, paramSets []map[string]string) (map[string]map[string]string, error) {
	if len(mergeKeys) < 1 {
		return nil, NoMergeKeys
	}

	deDuplicatedMergeKeys := make(map[string]bool, len(mergeKeys))
	for _, mergeKey := range mergeKeys {
		deDuplicatedMergeKeys[mergeKey] = false
	}

	paramSetsByMergeKey := make(map[string]map[string]string, len(paramSets))
	for _, paramSet := range paramSets {
		paramSetKey := make(map[string]string)
		for mergeKey, _ := range deDuplicatedMergeKeys {
			paramSetKey[mergeKey] = paramSet[mergeKey]
		}
		paramSetKeyJson, err := json.Marshal(paramSetKey)
		if err != nil {
			return nil, err
		}
		paramSetKeyString := string(paramSetKeyJson)
		if _, exists := paramSetsByMergeKey[paramSetKeyString]; exists {
			return nil, NonUniqueParamSets
		}
		paramSetsByMergeKey[paramSetKeyString] = paramSet
	}

	return paramSetsByMergeKey, nil
}

func (m *MergeGenerator) getParams(appSetBaseGenerator argoprojiov1alpha1.ApplicationSetNestedGenerator, appSet *argoprojiov1alpha1.ApplicationSet) ([]map[string]string, error) {
	var matrix *argoprojiov1alpha1.MatrixGenerator
	if appSetBaseGenerator.Matrix != nil {
		matrix = appSetBaseGenerator.Matrix.ToMatrixGenerator()
	}

	var mergeGenerator *argoprojiov1alpha1.MergeGenerator
	if appSetBaseGenerator.Merge != nil {
		mergeGenerator = appSetBaseGenerator.Merge.ToMergeGenerator()
	}

	t, err := Transform(
		argoprojiov1alpha1.ApplicationSetGenerator{
			List:                    appSetBaseGenerator.List,
			Clusters:                appSetBaseGenerator.Clusters,
			Git:                     appSetBaseGenerator.Git,
			SCMProvider:             appSetBaseGenerator.SCMProvider,
			ClusterDecisionResource: appSetBaseGenerator.ClusterDecisionResource,
			PullRequest:             appSetBaseGenerator.PullRequest,
			Matrix:                  matrix,
			Merge:                   mergeGenerator,
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

func (m *MergeGenerator) GetRequeueAfter(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) time.Duration {
	res := maxDuration
	var found bool

	for _, r := range appSetGenerator.Merge.Generators {
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

func (m *MergeGenerator) GetTemplate(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) *argoprojiov1alpha1.ApplicationSetTemplate {
	return &appSetGenerator.Merge.Template
}
