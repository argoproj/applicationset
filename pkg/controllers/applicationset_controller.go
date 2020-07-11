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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/kubernetes/pkg/apis/core"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	argov1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
)

// ApplicationSetReconciler reconciles a ApplicationSet object
type ApplicationSetReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=argoproj.io,resources=applicationsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=argoproj.io,resources=applicationsets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=argoproj.io,resources=applicationsets/status,verbs=get;update;patch

func (r *ApplicationSetReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = log.WithField("applicationset", req.NamespacedName)

	var applicationSetInfo argoprojiov1alpha1.ApplicationSet
	if err := r.Get(ctx, req.NamespacedName, &applicationSetInfo); err != nil {
		log.Infof("Unable to fetch applicationSetInfo %e", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	listGenerator := generators.NewListGenerator()
	clusterGenerator := generators.NewClusterGenerator(r.Client)

	// desiredApplications is the main list of all expected Applications from all generators in this appset.
	var desiredApplications []argov1alpha1.Application
	for _, tmpGenerator := range applicationSetInfo.Spec.Generators {
		var apps []argov1alpha1.Application
		var err error
		if tmpGenerator.List != nil {
			apps, err = listGenerator.GenerateApplications(&tmpGenerator, &applicationSetInfo)
		} else if tmpGenerator.Clusters != nil {
			apps, err = clusterGenerator.GenerateApplications(&tmpGenerator, &applicationSetInfo)
		}
		log.Infof("apps from generator: %+v", apps)
		if err != nil {
			log.WithError(err).Error("error generating applications")
		}
		desiredApplications = append(desiredApplications, apps...)

	}

	if err := r.applyApplicationsToCluster(ctx, applicationSetInfo, desiredApplications); err != nil {
		log.Infof("Unable to create applications applicationSetInfo %e", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ApplicationSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(&argov1alpha1.Application{}, ".metadata.controller", func(rawObj runtime.Object) []string {
		// grab the job object, extract the owner...
		app := rawObj.(*argov1alpha1.Application)
		owner := metav1.GetControllerOf(app)
		if owner == nil {
			return nil
		}
		// ...make sure it's a application set...
		if owner.APIVersion != argoprojiov1alpha1.GroupVersion.String() || owner.Kind != "ApplicationSet" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&argoprojiov1alpha1.ApplicationSet{}).
		Owns(&argov1alpha1.Application{}).
		Watches(
			&source.Kind{Type: &corev1.Secret{}},
			&clusterSecretEventHandler{
				Client: mgr.GetClient(),
				Log:    log.WithField("type", "createSecretEventHandler"),
			}).
		// TODO: also watch Applications and respond on changes if we own them.
		Complete(r)
}

func (r *ApplicationSetReconciler) applyApplicationsToCluster(ctx context.Context, applicationSetInfo argoprojiov1alpha1.ApplicationSet, appList []argov1alpha1.Application) error {

	// Save current applications to be able to delete the ones that are not in appList
	var current argov1alpha1.ApplicationList
	_ = r.Client.List(context.Background(), &current, client.MatchingFields{".metadata.controller": applicationSetInfo.Name})

	m := make(map[string]bool) // Will holds the app names in appList for the deletion process

	//create or updates the application in appList
	for _, app := range appList {
		m[app.Name] = true
		app.Namespace = applicationSetInfo.Namespace

		found := app
		action, err := ctrl.CreateOrUpdate(ctx, r.Client, &found, func() error {
			found.Spec = app.Spec
			return controllerutil.SetControllerReference(&applicationSetInfo, &found, r.Scheme)
		})

		if err != nil {
			log.Error(err, fmt.Sprintf("failed to CreateOrUpdate Application %s resource for applicationSet %s", app.Name, applicationSetInfo.Name))
			continue
		}

		r.Recorder.Eventf(&applicationSetInfo, core.EventTypeNormal, fmt.Sprint(action), "%s Application %q", action, app.Name)
		log.Infof("%s Application %s resource for applicationSet %s", action, app.Name, applicationSetInfo.Name)
	}

	// Delete apps that are not in m[string]bool
	for _, app := range current.Items {
		_, exists := m[app.Name]

		if exists == false {
			err := r.Client.Delete(ctx, &app)
			if err != nil {
				log.Error(err, fmt.Sprintf("failed to delete Application %s resource for applicationSet %s", app.Name, applicationSetInfo.Name))
				continue
			}
			r.Recorder.Eventf(&applicationSetInfo, core.EventTypeNormal, "Deleted", "Deleted Application %q", app.Name)
			log.Infof("Deleted Application %s resource for applicationSet %s", app.Name, applicationSetInfo.Name)
		}
	}

	return nil

}

func (r *ApplicationSetReconciler) getApplications(ctx context.Context, applicationSetInfo argoprojiov1alpha1.ApplicationSet) ([]argov1alpha1.Application, error) {

	return nil, nil

}

var _ handler.EventHandler = &clusterSecretEventHandler{}
