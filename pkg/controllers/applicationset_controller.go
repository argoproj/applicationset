/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"github.com/argoproj-labs/applicationset/pkg/generators"
	argov1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"k8s.io/client-go/tools/record"
	"k8s.io/kubernetes/pkg/apis/core"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
)

// ApplicationSetReconciler reconciles a ApplicationSet object
type ApplicationSetReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	RepoServerAddr string
}

// +kubebuilder:rbac:groups=argoproj.io,resources=applicationsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=argoproj.io,resources=applicationsets/status,verbs=get;update;patch

func (r *ApplicationSetReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = log.WithField("applicationset", req.NamespacedName)

	var applicationSetInfo argoprojiov1alpha1.ApplicationSet
	if err := r.Get(ctx, req.NamespacedName, &applicationSetInfo); err != nil {
		log.Info("Unable to fetch applicationSetInfo %v", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

		var listGenerator generators.Generator
		listGenerator = generators.NewListGenerator()

		var gitGenerator generators.Generator
		gitGenerator = generators.NewGitGenerator(r.RepoServerAddr)

		var appList []argov1alpha1.Application
		for _, tmpGenerator := range applicationSetInfo.Spec.Generators {
			if tmpGenerator.List != nil {
				newApplications, err :=  listGenerator.GenerateApplications(&applicationSetInfo)
				log.Infof("newApplications %++v error %++v", newApplications, err)
				appList = append(appList, newApplications...)
			}
			if tmpGenerator.Git != nil {
				newApplications, err :=  gitGenerator.GenerateApplications(&applicationSetInfo)
				log.Infof("newApplications %++v error %++v", newApplications, err)
				appList = append(appList, newApplications...)
			}
		}

		if err := r.createApplications(ctx, applicationSetInfo, appList); err != nil {
			log.Info("Unable to create applications applicationSetInfo %v", err)
			return ctrl.Result{}, err
		}

	return ctrl.Result{}, nil
}

func (r *ApplicationSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&argoprojiov1alpha1.ApplicationSet{}).
		Complete(r)
}

func (r *ApplicationSetReconciler) createApplications(ctx context.Context, applicationSetInfo argoprojiov1alpha1.ApplicationSet, appList []argov1alpha1.Application) error {

	for _, app := range appList {
		app.Namespace = applicationSetInfo.Namespace
		if err := r.Client.Create(ctx, &app); err != nil {
			log.Error(err, fmt.Sprintf("failed to create Application %s resource for applicationSet %s", app.Name, applicationSetInfo.Name))
			continue
		}

		r.Recorder.Eventf(&applicationSetInfo, core.EventTypeNormal, "Created", "Created Application %q", app.Name)
		log.Infof("created Application %s resource for applicationSet %s", app.Name, applicationSetInfo.Name)
	}

	return nil

}
