package generators

import (
	"fmt"
	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/argoproj-labs/applicationset/pkg/utils"
	argov1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	log "github.com/sirupsen/logrus"
)

var _ Generator = (*ListGenerator)(nil)

type ListGenerator struct {
}

func NewListGenerator() Generator {
	g := &ListGenerator{}
	return g
}

func (g *ListGenerator) GenerateApplications(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator,
	appSet *argoprojiov1alpha1.ApplicationSet) ([]argov1alpha1.Application, error) {
	if appSetGenerator == nil || appSet == nil {
		return nil, fmt.Errorf("ApplicationSet is empty")
	}

	if appSetGenerator.List == nil {
		return nil, nil
	}

	listGenerator := appSetGenerator.List

	if listGenerator == nil {
		return nil, fmt.Errorf("There isn't list generator ")
	}

	var tmplApplication argov1alpha1.Application
	tmplApplication.Namespace = appSet.Spec.Template.Namespace
	tmplApplication.Name = appSet.Spec.Template.Name
	tmplApplication.Spec = appSet.Spec.Template.Spec

	params := make(map[string]string, 2)
	for _, tmpItem := range listGenerator.Elements {
		params[utils.ClusterListGeneratorKeyName] = tmpItem.Cluster
		params[utils.UrlGeneratorKeyName] = tmpItem.Url
		tmpApplication, err := utils.RenderTemplateParams(&tmplApplication, params)
		log.Infof("tmpApplication %++v", tmpApplication)
		log.Infof("error %v", err)
	}
	return nil, nil
}
