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
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/argoproj-labs/applicationset/pkg/generators"
	"github.com/argoproj-labs/applicationset/pkg/utils"
	argov1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/go-logr/logr"
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

	"github.com/imdario/mergo"
)

// ApplicationSetReconciler reconciles a ApplicationSet object
type ApplicationSetReconciler struct {
	client.Client
	Log        logr.Logger
	Scheme     *runtime.Scheme
	Recorder   record.EventRecorder
	Generators map[string]generators.Generator
	utils.Policy
	utils.Renderer
}

// +kubebuilder:rbac:groups=argoproj.io,resources=applicationsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=argoproj.io,resources=applicationsets/status,verbs=get;update;patch

func (r *ApplicationSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("applicationset", req.NamespacedName)
	_ = log.WithField("applicationset", req.NamespacedName)

	var applicationSetInfo argoprojiov1alpha1.ApplicationSet
	if err := r.Get(ctx, req.NamespacedName, &applicationSetInfo); err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.WithField("request", req).WithError(err).Infof("unable to get ApplicationSet")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Log a warning if there are unrecognized generators
	checkInvalidGenerators(&applicationSetInfo)

	// desiredApplications is the main list of all expected Applications from all generators in this appset.
	desiredApplications, err := r.generateApplications(applicationSetInfo)
	if err != nil {
		return ctrl.Result{}, err
	}
	if hasDuplicates, name := hasDuplicateNames(desiredApplications); hasDuplicates {
		// The reconciler presumes that any errors that are returned are a signal
		// that the resource should attempt to be reconciled again (causing
		// Reconcile to be called again, which will return the same error, ad
		// infinitum until we are exponentially backed off).
		//
		// In this case, since we know that what the user provided is incorrect
		// (we successfully generated and templated their ApplicationSet, but the
		// result of that was bad), no matter how many times we try to do so it
		// will fail. So just log it and return that the resource was
		// successfully reconciled (which is true... it was reconciled to an
		// error condition).
		log.Errorf("ApplicationSet %s contains applications with duplicate name: %s", applicationSetInfo.Name, name)
		return ctrl.Result{}, nil
	}

	if r.Policy.Update() {
		err = r.createOrUpdateInCluster(ctx, applicationSetInfo, desiredApplications)
		if err != nil {
			return ctrl.Result{}, err
		}
	} else {
		err = r.createInCluster(ctx, applicationSetInfo, desiredApplications)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if r.Policy.Delete() {
		err = r.deleteInCluster(ctx, applicationSetInfo, desiredApplications)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	requeueAfter := r.getMinRequeueAfter(&applicationSetInfo)
	log.WithField("requeueAfter", requeueAfter).Info("end reconcile")

	return ctrl.Result{
		RequeueAfter: requeueAfter,
	}, nil
}

// Log a warning if there are unrecognized generators
func checkInvalidGenerators(applicationSetInfo *argoprojiov1alpha1.ApplicationSet) {
	hasInvalidGenerators, invalidGenerators := invalidGenerators(applicationSetInfo)
	if len(invalidGenerators) > 0 {
		gnames := []string{}
		for n := range invalidGenerators {
			gnames = append(gnames, n)
		}
		sort.Strings(gnames)
		aname := applicationSetInfo.ObjectMeta.Name
		msg := "ApplicationSet %s contains unrecognized generators: %s"
		log.Warnf(msg, aname, strings.Join(gnames, ", "))
	} else if hasInvalidGenerators {
		name := applicationSetInfo.ObjectMeta.Name
		msg := "ApplicationSet %s contains unrecognized generators"
		log.Warnf(msg, name)
	}
}

// Return true if there are unknown generators specified in the application set.  If we can discover the names
// of these generators, return the names as the keys in a map
func invalidGenerators(applicationSetInfo *argoprojiov1alpha1.ApplicationSet) (bool, map[string]bool) {
	names := make(map[string]bool)
	hasInvalidGenerators := false
	for index, generator := range applicationSetInfo.Spec.Generators {
		v := reflect.Indirect(reflect.ValueOf(generator))
		found := false
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			if !field.CanInterface() {
				continue
			}
			if !reflect.ValueOf(field.Interface()).IsNil() {
				found = true
				break
			}
		}
		if !found {
			hasInvalidGenerators = true
			addInvalidGeneratorNames(names, applicationSetInfo, index)
		}
	}
	return hasInvalidGenerators, names
}

func addInvalidGeneratorNames(names map[string]bool, applicationSetInfo *argoprojiov1alpha1.ApplicationSet, index int) {
	// The generator names are stored in the "kubectl.kubernetes.io/last-applied-configuration" annotation
	config := applicationSetInfo.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"]
	var values map[string]interface{}
	err := json.Unmarshal([]byte(config), &values)
	if err != nil {
		log.Warnf("couldn't unmarshal kubectl.kubernetes.io/last-applied-configuration: %+v", config)
		return
	}

	spec, ok := values["spec"].(map[string]interface{})
	if !ok {
		log.Warn("coundn't get spec from kubectl.kubernetes.io/last-applied-configuration annotation")
		return
	}

	generators, ok := spec["generators"].([]interface{})
	if !ok {
		log.Warn("coundn't get generators from kubectl.kubernetes.io/last-applied-configuration annotation")
		return
	}

	if index >= len(generators) {
		log.Warnf("index %d out of range %d for generator in kubectl.kubernetes.io/last-applied-configuration", index, len(generators))
		return
	}

	generator, ok := generators[index].(map[string]interface{})
	if !ok {
		log.Warn("coundn't get generator from kubectl.kubernetes.io/last-applied-configuration annotation")
		return
	}

	for key := range generator {
		names[key] = true
		break
	}
}

func hasDuplicateNames(applications []argov1alpha1.Application) (bool, string) {
	nameSet := map[string]struct{}{}
	for _, app := range applications {
		if _, present := nameSet[app.Name]; present {
			return true, app.Name
		}
		nameSet[app.Name] = struct{}{}
	}
	return false, ""
}

func (r *ApplicationSetReconciler) GetRelevantGenerators(requestedGenerator *argoprojiov1alpha1.ApplicationSetGenerator) []generators.Generator {
	var res []generators.Generator

	v := reflect.Indirect(reflect.ValueOf(requestedGenerator))
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.CanInterface() {
			continue
		}

		if !reflect.ValueOf(field.Interface()).IsNil() {
			res = append(res, r.Generators[v.Type().Field(i).Name])
		}
	}

	return res
}

func (r *ApplicationSetReconciler) getMinRequeueAfter(applicationSetInfo *argoprojiov1alpha1.ApplicationSet) time.Duration {
	var res time.Duration
	for _, requestedGenerator := range applicationSetInfo.Spec.Generators {

		generators := r.GetRelevantGenerators(&requestedGenerator)

		for _, g := range generators {
			t := g.GetRequeueAfter(&requestedGenerator)

			if res == 0 {
				res = t
			} else if t != 0 && t < res {
				res = t
			}
		}
	}

	return res
}

func getTempApplication(applicationSetTemplate argoprojiov1alpha1.ApplicationSetTemplate) *argov1alpha1.Application {
	var tmplApplication argov1alpha1.Application
	tmplApplication.Annotations = applicationSetTemplate.Annotations
	tmplApplication.Labels = applicationSetTemplate.Labels
	tmplApplication.Namespace = applicationSetTemplate.Namespace
	tmplApplication.Name = applicationSetTemplate.Name
	tmplApplication.Spec = applicationSetTemplate.Spec

	return &tmplApplication
}

func mergeGeneratorTemplate(g generators.Generator, requestedGenerator *argoprojiov1alpha1.ApplicationSetGenerator,  applicationSetTemplate argoprojiov1alpha1.ApplicationSetTemplate) argoprojiov1alpha1.ApplicationSetTemplate{
	dest := g.GetTemplate(requestedGenerator)
	_ = mergo.Merge(dest, applicationSetTemplate)

	return *dest
}


func (r *ApplicationSetReconciler) generateApplications(applicationSetInfo argoprojiov1alpha1.ApplicationSet) ([]argov1alpha1.Application, error) {
	res := []argov1alpha1.Application{}

	var firstError error
	for _, requestedGenerator := range applicationSetInfo.Spec.Generators {
		generators := r.GetRelevantGenerators(&requestedGenerator)
		for _, g := range generators {
			params, err := g.GenerateParams(&requestedGenerator)
			if err != nil {
				log.WithError(err).WithField("generator", g).
					Error("error generating params")
				if firstError == nil {
					firstError = err
				}
				continue
			}

			tmplApplication := getTempApplication(mergeGeneratorTemplate(g, &requestedGenerator, applicationSetInfo.Spec.Template))

			for _, p := range params {
				app, err := r.Renderer.RenderTemplateParams(tmplApplication, p)
				if err != nil {
					log.WithError(err).WithField("params", params).WithField("generator", g).
						Error("error generating application from params")
					if firstError == nil {
						firstError = err
					}
					continue
				}
				res = append(res, *app)
			}

			log.WithField("generator", g).Infof("generated %d applications", len(res))
			log.WithField("generator", g).Debugf("apps from generator: %+v", res)

		}
	}
	return res, firstError
}

func (r *ApplicationSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.TODO(), &argov1alpha1.Application{}, ".metadata.controller", func(rawObj client.Object) []string {
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

// createOrUpdateInCluster will create / update application resources in the cluster.
// For new application it will call create
// For application that need to update it will call update
// The function also adds owner reference to all applications, and uses it for delete them.
func (r *ApplicationSetReconciler) createOrUpdateInCluster(ctx context.Context, applicationSet argoprojiov1alpha1.ApplicationSet, desiredApplications []argov1alpha1.Application) error {

	var firstError error
	//create or updates the application in appList
	for _, app := range desiredApplications {
		appLog := log.WithFields(log.Fields{"app": app.Name, "appSet": applicationSet.Name})
		app.Namespace = applicationSet.Namespace

		found := app
		action, err := utils.CreateOrUpdate(ctx, r.Client, &found, func() error {
			found.Spec = app.Spec
			return controllerutil.SetControllerReference(&applicationSet, &found, r.Scheme)
		})

		if err != nil {
			appLog.WithError(err).WithField("action", action).Errorf("failed to %s Application", action)
			if firstError == nil {
				firstError = err
			}
			continue
		}

		r.Recorder.Eventf(&applicationSet, core.EventTypeNormal, fmt.Sprint(action), "%s Application %q", action, app.Name)
		appLog.Logf(log.InfoLevel, "%s Application", action)
	}
	return firstError
}

// createInCluster will filter from the desiredApplications only the application that needs to be created
// Then it will call createOrUpdateInCluster to do the actual create
func (r *ApplicationSetReconciler) createInCluster(ctx context.Context, applicationSet argoprojiov1alpha1.ApplicationSet, desiredApplications []argov1alpha1.Application) error {

	var createApps []argov1alpha1.Application
	current, err := r.getCurrentApplications(ctx, applicationSet)
	if err != nil {
		return err
	}

	m := make(map[string]bool) // Will holds the app names that are current in the cluster

	for _, app := range current {
		m[app.Name] = true
	}

	// filter applications that are not in m[string]bool (new to the cluster)
	for _, app := range desiredApplications {
		_, exists := m[app.Name]

		if !exists {
			createApps = append(createApps, app)
		}
	}

	return r.createOrUpdateInCluster(ctx, applicationSet, createApps)
}

func (r *ApplicationSetReconciler) getCurrentApplications(_ context.Context, applicationSet argoprojiov1alpha1.ApplicationSet) ([]argov1alpha1.Application, error) {
	// TODO: Should this use the context param?
	var current argov1alpha1.ApplicationList
	err := r.Client.List(context.Background(), &current, client.MatchingFields{".metadata.controller": applicationSet.Name})

	if err != nil {
		return nil, err
	}

	return current.Items, nil
}

// deleteInCluster will delete application that are current in the cluster but not in appList.
// The function must be called after all generators had been called and generated applications
func (r *ApplicationSetReconciler) deleteInCluster(ctx context.Context, applicationSet argoprojiov1alpha1.ApplicationSet, desiredApplications []argov1alpha1.Application) error {

	// Save current applications to be able to delete the ones that are not in appList
	current, err := r.getCurrentApplications(ctx, applicationSet)
	if err != nil {
		return err
	}

	m := make(map[string]bool) // Will holds the app names in appList for the deletion process

	for _, app := range desiredApplications {
		m[app.Name] = true
	}

	// Delete apps that are not in m[string]bool
	var firstError error
	for _, app := range current {
		appLog := log.WithFields(log.Fields{"app": app.Name, "appSet": applicationSet.Name})
		_, exists := m[app.Name]

		if !exists {
			err := r.Client.Delete(ctx, &app)
			if err != nil {
				appLog.WithError(err).Error("failed to delete Application")
				if firstError != nil {
					firstError = err
				}
				continue
			}
			r.Recorder.Eventf(&applicationSet, core.EventTypeNormal, "Deleted", "Deleted Application %q", app.Name)
			appLog.Log(log.InfoLevel, "Deleted application")
		}
	}
	return firstError
}

var _ handler.EventHandler = &clusterSecretEventHandler{}
