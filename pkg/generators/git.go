package generators

import (
	"fmt"
	"path"

	"github.com/argoproj-labs/applicationset/pkg/utils"

	log "github.com/sirupsen/logrus"
	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	argov1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"

)

var _ Generator = (*GitGenerator)(nil)

type GitGenerator struct {
}

func NewGitGenerator() Generator {
	g := &GitGenerator{}
	return g
}

func (g *GitGenerator) GenerateApplications(appSet *argoprojiov1alpha1.ApplicationSet) ([]argov1alpha1.Application, error) {
	if appSet == nil {
		return nil, fmt.Errorf("ApplicationSet is empty")
	}

	var GitGenerator *argoprojiov1alpha1.GitGenerator

	for _, tmpGenerator := range appSet.Spec.Generators {
		if tmpGenerator.Git != nil {
			GitGenerator = tmpGenerator.Git
			break
		}
	}

	if GitGenerator == nil {
		return nil, fmt.Errorf("There isn't git generator ")
	}



	if len(GitGenerator.Directories) > 0 {
		//git get all directories & filter by paths
		var pathes []string

		for _, p := range pathes {

			var tmplApplication argov1alpha1.Application
			tmplApplication.Namespace = appSet.Spec.Template.Namespace
			tmplApplication.Name = appSet.Spec.Template.Name
			tmplApplication.Spec = appSet.Spec.Template.Spec

			params := make(map[string]string, 2)
			params["path"] = p
			params["path.basename"] = path.Base(p)

			tmpApplication, err := utils.RenderTemplateParams(&tmplApplication, params)
			log.Infof("tmpApplication %++v", tmpApplication)
			log.Infof("error %v", err)

		}


	}

	return nil, nil
}
