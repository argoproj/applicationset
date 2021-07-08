package applicationsets

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/argoproj-labs/applicationset/test/e2e/fixture/applicationsets/utils"
	argocommon "github.com/argoproj/argo-cd/v2/common"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// this implements the "when" part of given/when/then
//
// none of the func implement error checks, and that is complete intended, you should check for errors
// using the Then()
type Actions struct {
	context        *Context
	lastOutput     string
	lastError      error
	describeAction string
	ignoreErrors   bool
}

var pdGVR = schema.GroupVersionResource{
	Group:    "cluster.open-cluster-management.io",
	Version:  "v1alpha1",
	Resource: "placementdecisions",
}

// IgnoreErrors sets whether to ignore
func (a *Actions) IgnoreErrors() *Actions {
	a.ignoreErrors = true
	return a
}

func (a *Actions) DoNotIgnoreErrors() *Actions {
	a.ignoreErrors = false
	return a
}

func (a *Actions) And(block func()) *Actions {
	a.context.t.Helper()
	block()
	return a
}

func (a *Actions) Then() *Consequences {
	a.context.t.Helper()
	return &Consequences{a.context, a}
}

// CreateClusterSecret creates a faux cluster secret, with the given cluster server and cluster name (this cluster
// will not actually be used by the Argo CD controller, but that's not needed for our E2E tests)
func (a *Actions) CreateClusterSecret(secretName string, clusterName string, clusterServer string) *Actions {

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: utils.ArgoCDNamespace,
			Labels: map[string]string{
				argocommon.LabelKeySecretType: argocommon.LabelValueSecretTypeCluster,
				utils.TestingLabel:            "true",
			},
		},
		Data: map[string][]byte{
			"name":   []byte(clusterName),
			"server": []byte(clusterServer),
			"config": []byte("{\"username\":\"foo\",\"password\":\"foo\"}"),
		},
	}

	fixtureClient := utils.GetE2EFixtureK8sClient()
	_, err := fixtureClient.KubeClientset.CoreV1().Secrets(secret.Namespace).Create(context.Background(), secret, metav1.CreateOptions{})

	a.describeAction = fmt.Sprintf("creating cluster Secret '%s'", secretName)
	a.lastOutput, a.lastError = "", err
	a.verifyAction()

	return a
}

// DeleteClusterSecret deletes a faux cluster secret
func (a *Actions) DeleteClusterSecret(secretName string) *Actions {

	err := utils.GetE2EFixtureK8sClient().KubeClientset.CoreV1().Secrets(utils.ArgoCDNamespace).Delete(context.Background(), secretName, metav1.DeleteOptions{})

	a.describeAction = fmt.Sprintf("deleting cluster Secret '%s'", secretName)
	a.lastOutput, a.lastError = "", err
	a.verifyAction()

	return a
}

// DeleteConfigMap deletes a faux cluster secret
func (a *Actions) DeleteConfigMap(configMapName string) *Actions {

	err := utils.GetE2EFixtureK8sClient().KubeClientset.CoreV1().ConfigMaps(utils.ArgoCDNamespace).Delete(context.Background(), configMapName, metav1.DeleteOptions{})

	a.describeAction = fmt.Sprintf("deleting configMap '%s'", configMapName)
	a.lastOutput, a.lastError = "", err
	a.verifyAction()

	return a
}

// DeletePlacementDecision deletes a faux cluster secret
func (a *Actions) DeletePlacementDecision(placementDecisionName string) *Actions {

	err := utils.GetE2EFixtureK8sClient().DynamicClientset.Resource(pdGVR).Namespace(utils.ArgoCDNamespace).Delete(context.Background(), placementDecisionName, metav1.DeleteOptions{})

	a.describeAction = fmt.Sprintf("deleting placement decision '%s'", placementDecisionName)
	a.lastOutput, a.lastError = "", err
	a.verifyAction()

	return a
}

// Create a temporary namespace, from utils.ApplicationSet, for use by the test.
// This namespace will be deleted on subsequent tests.
func (a *Actions) CreateNamespace() *Actions {
	a.context.t.Helper()

	fixtureClient := utils.GetE2EFixtureK8sClient()

	_, err := fixtureClient.KubeClientset.CoreV1().Namespaces().Create(context.Background(),
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: utils.ApplicationSetNamespace}}, metav1.CreateOptions{})

	a.describeAction = fmt.Sprintf("creating namespace '%s'", utils.ApplicationSetNamespace)
	a.lastOutput, a.lastError = "", err
	a.verifyAction()

	return a
}

// Create creates an ApplicationSet using the provided value
func (a *Actions) Create(appSet v1alpha1.ApplicationSet) *Actions {
	a.context.t.Helper()

	appSet.APIVersion = "argoproj.io/v1alpha1"
	appSet.Kind = "ApplicationSet"

	fixtureClient := utils.GetE2EFixtureK8sClient()
	newResource, err := fixtureClient.AppSetClientset.Create(context.Background(), utils.MustToUnstructured(&appSet), metav1.CreateOptions{})

	if err == nil {
		a.context.name = newResource.GetName()
	}

	a.describeAction = fmt.Sprintf("creating ApplicationSet '%s'", appSet.Name)
	a.lastOutput, a.lastError = "", err
	a.verifyAction()

	return a
}

// Create a ConfigMap for the ClusterResourceList generator
func (a *Actions) CreatePlacementDecisionConfigMap(configMapName string) *Actions {
	a.context.t.Helper()

	fixtureClient := utils.GetE2EFixtureK8sClient()

	_, err := fixtureClient.KubeClientset.CoreV1().ConfigMaps(utils.ArgoCDNamespace).Get(context.Background(), configMapName, metav1.GetOptions{})

	// Don't do anything if it exists
	if err == nil {
		return a
	}

	_, err = fixtureClient.KubeClientset.CoreV1().ConfigMaps(utils.ArgoCDNamespace).Create(context.Background(),
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: configMapName,
			},
			Data: map[string]string{
				"apiVersion":    "cluster.open-cluster-management.io/v1alpha1",
				"kind":          "placementdecisions",
				"statusListKey": "decisions",
				"matchKey":      "clusterName",
			},
		}, metav1.CreateOptions{})

	a.describeAction = fmt.Sprintf("creating configmap '%s'", configMapName)
	a.lastOutput, a.lastError = "", err
	a.verifyAction()

	return a
}

func (a *Actions) CreatePlacementDecision(placementDecisionName string) *Actions {
	a.context.t.Helper()

	fixtureClient := utils.GetE2EFixtureK8sClient().DynamicClientset

	_, err := fixtureClient.Resource(pdGVR).Namespace(utils.ArgoCDNamespace).Get(
		context.Background(),
		placementDecisionName,
		metav1.GetOptions{})
	// If already exists
	if err == nil {
		return a
	}

	placementDecision := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      placementDecisionName,
				"namespace": utils.ArgoCDNamespace,
			},
			"kind":       "PlacementDecision",
			"apiVersion": "cluster.open-cluster-management.io/v1alpha1",
			"status":     map[string]interface{}{},
		},
	}

	_, err = fixtureClient.Resource(pdGVR).Namespace(utils.ArgoCDNamespace).Create(
		context.Background(),
		placementDecision,
		metav1.CreateOptions{})

	a.describeAction = fmt.Sprintf("creating placementDecision '%v'", placementDecisionName)
	a.lastOutput, a.lastError = "", err
	a.verifyAction()

	return a
}

func (a *Actions) StatusUpdatePlacementDecision(placementDecisionName string, clusterList []interface{}) *Actions {
	a.context.t.Helper()

	fixtureClient := utils.GetE2EFixtureK8sClient().DynamicClientset
	placementDecision, err := fixtureClient.Resource(pdGVR).Namespace(utils.ArgoCDNamespace).Get(
		context.Background(),
		placementDecisionName,
		metav1.GetOptions{})

	placementDecision.Object["status"] = map[string]interface{}{
		"decisions": clusterList,
	}

	if err == nil {
		_, err = fixtureClient.Resource(pdGVR).Namespace(utils.ArgoCDNamespace).UpdateStatus(
			context.Background(),
			placementDecision,
			metav1.UpdateOptions{})
	}
	a.describeAction = fmt.Sprintf("status update placementDecision for '%v'", clusterList)
	a.lastOutput, a.lastError = "", err
	a.verifyAction()

	return a
}

// Delete deletes the ApplicationSet within the context
func (a *Actions) Delete() *Actions {
	a.context.t.Helper()

	fixtureClient := utils.GetE2EFixtureK8sClient()

	deleteProp := metav1.DeletePropagationForeground
	err := fixtureClient.AppSetClientset.Delete(context.Background(), a.context.name, metav1.DeleteOptions{PropagationPolicy: &deleteProp})
	a.describeAction = fmt.Sprintf("Deleting ApplicationSet '%s' %v", a.context.name, err)
	a.lastOutput, a.lastError = "", err
	a.verifyAction()

	return a
}

// get retrieves the ApplicationSet (by name) that was created by an earlier Create action
func (a *Actions) get() (*v1alpha1.ApplicationSet, error) {
	appSet := v1alpha1.ApplicationSet{}

	fixtureClient := utils.GetE2EFixtureK8sClient()
	newResource, err := fixtureClient.AppSetClientset.Get(context.Background(), a.context.name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	bytes, err := newResource.MarshalJSON()
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(bytes, &appSet)
	if err != nil {
		return nil, err
	}

	return &appSet, nil

}

// Update retrieves the latest copy the ApplicationSet, then allows the caller to mutate it via 'toUpdate', with
// the result applied back to the cluster resource
func (a *Actions) Update(toUpdate func(*v1alpha1.ApplicationSet)) *Actions {
	a.context.t.Helper()

	appSet, err := a.get()
	if err == nil {
		toUpdate(appSet)
		a.describeAction = fmt.Sprintf("updating ApplicationSet '%s'", appSet.Name)

		fixtureClient := utils.GetE2EFixtureK8sClient()
		_, err = fixtureClient.AppSetClientset.Update(context.Background(), utils.MustToUnstructured(&appSet), metav1.UpdateOptions{})
	}
	a.lastOutput, a.lastError = "", err
	a.verifyAction()

	return a
}

func (a *Actions) verifyAction() {
	a.context.t.Helper()

	if a.describeAction != "" {
		log.Infof("action: %s", a.describeAction)
		a.describeAction = ""
	}

	if !a.ignoreErrors {
		a.Then().Expect(Success(""))
	}

}
