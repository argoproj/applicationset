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
	"github.com/argoproj-labs/applicationset/pkg/refresher"
	"github.com/argoproj-labs/applicationset/pkg/services"
	argov1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/kubernetes/pkg/apis/core"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
)

// ApplicationSetReconciler reconciles a ApplicationSet object
type ApplicationSetReconciler struct {
	client.Client
	Scheme         	*runtime.Scheme
	Recorder       	record.EventRecorder
	RepoServerAddr 	string
	AppsService		services.Apps
	Refresher		refresher.Refresher
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
	GitGenerator := generators.NewGitGenerator(r.AppsService)

	// desiredApplications is the main list of all expected Applications from all generators in this appset.
	var desiredApplications []argov1alpha1.Application

	for _, tmpGenerator := range applicationSetInfo.Spec.Generators {
		var apps []argov1alpha1.Application
		var err error
		if tmpGenerator.List != nil {
			apps, err = listGenerator.GenerateApplications(&tmpGenerator, &applicationSetInfo)
		} else if tmpGenerator.Clusters != nil {
			apps, err = clusterGenerator.GenerateApplications(&tmpGenerator, &applicationSetInfo)
		} else if tmpGenerator.Git != nil {
			apps, err = GitGenerator.GenerateApplications(&tmpGenerator, &applicationSetInfo)
			r.Refresher.Add(req)
		}
		log.Infof("apps from generator: %+v", apps)
		if err != nil {
			log.WithError(err).Error("error generating applications")
			continue
		}
		
		desiredApplications = append(desiredApplications, apps...)

	}

	r.createOrUpdateInCluster(ctx, applicationSetInfo, desiredApplications)
	r.deleteInCluster(ctx, applicationSetInfo, desiredApplications)

	return ctrl.Result{}, nil
}

func (r *ApplicationSetReconciler) SetupWithManager(mgr ctrl.Manager, events chan event.GenericEvent) error {
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
		Watches(
		&source.Channel{Source: events},
		&handler.EnqueueRequestForObject{},
		).
		// TODO: also watch Applications and respond on changes if we own them.
		Complete(r)
}

// createOrUpdateInCluster will create / update application resources in the cluster.
// For new application it will call create
// For application that need to update it will call update
// The function also adds owner reference to all applications, and uses it for delete them.
func (r *ApplicationSetReconciler) createOrUpdateInCluster(ctx context.Context, applicationSet argoprojiov1alpha1.ApplicationSet, desiredApplications []argov1alpha1.Application) {

	//create or updates the application in appList
	for _, app := range desiredApplications {
		appLog := log.WithFields(log.Fields{"app": app.Name, "appSet": applicationSet.Name})
		app.Namespace = applicationSet.Namespace

		found := app
		action, err := ctrl.CreateOrUpdate(ctx, r.Client, &found, func() error {
			found.Spec = app.Spec
			return controllerutil.SetControllerReference(&applicationSet, &found, r.Scheme)
		})

		if err != nil {
			appLog.WithError(err).Errorf("failed to %s Application", action)
			continue
		}

		r.Recorder.Eventf(&applicationSet, core.EventTypeNormal, fmt.Sprint(action), "%s Application %q", action, app.Name)
		appLog.Logf(log.InfoLevel, "%s Application", action)
	}
}

// deleteInCluster will delete application that are current in the cluster but not in appList.
// The function must be called after all generators had been called and generated applications
func (r *ApplicationSetReconciler) deleteInCluster(ctx context.Context, applicationSet argoprojiov1alpha1.ApplicationSet, desiredApplications []argov1alpha1.Application) {

	// Save current applications to be able to delete the ones that are not in appList
	var current argov1alpha1.ApplicationList
	_ = r.Client.List(context.Background(), &current, client.MatchingFields{".metadata.controller": applicationSet.Name})

	m := make(map[string]bool) // Will holds the app names in appList for the deletion process

	for _, app := range desiredApplications {
		m[app.Name] = true
	}

	// Delete apps that are not in m[string]bool
	for _, app := range current.Items {
		appLog := log.WithFields(log.Fields{"app": app.Name, "appSet": applicationSet.Name})
		_, exists := m[app.Name]

		if exists == false {
			err := r.Client.Delete(ctx, &app)
			if err != nil {
				appLog.WithError(err).Error("failed to delete Application")
				continue
			}
			r.Recorder.Eventf(&applicationSet, core.EventTypeNormal, "Deleted", "Deleted Application %q", app.Name)
			appLog.Log(log.InfoLevel, "Deleted application")
		}
	}
}

var _ handler.EventHandler = &clusterSecretEventHandler{}
