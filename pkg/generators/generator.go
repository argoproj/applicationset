package generators

import (
	"fmt"
	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	//log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Generator struct {
	commonClient client.Client
}

func NewGenerator(client client.Client) *Generator {
	g := &Generator{
		commonClient: client,
	}
	return g
}

func (g* Generator) CreateApplicationByItem(appSet *argoprojiov1alpha1.ApplicationSet) error {
	if appSet == nil {
		return fmt.Errorf("ApplicationSet is empty ")
	}

	return nil
}
