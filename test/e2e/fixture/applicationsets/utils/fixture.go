package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/argoproj-labs/applicationset/api/v1alpha1"
	log "github.com/sirupsen/logrus"

	appclientset "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned"
	"k8s.io/apimachinery/pkg/api/equality"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// ArgoCDNamespace is the namespace into which Argo CD and ApplicationSet controller are deployed,
	// and in which Application resources should be created.
	ArgoCDNamespace = "argocd-e2e"

	// ApplicationSetNamespace is the namespace into which temporary resources (such as Deployments/Pods/etc)
	// can be deployed, such as using it as the target namespace in an Application resource.
	// Note: this is NOT the namespace the ApplicationSet controller is deployed to; see ArgoCDNamespace.
	ApplicationSetNamespace = "applicationset-e2e"

	TmpDir       = "/tmp/applicationset-e2e"
	TestingLabel = "e2e.argoproj.io"
)

var (
	id string

	// call GetClientVars() to retrieve the Kubernetes client data for E2E test fixtures
	clientInitialized  sync.Once
	internalClientVars *E2EFixtureK8sClient
)

// E2EFixtureK8sClient contains Kubernetes clients initialized from local k8s configuration
type E2EFixtureK8sClient struct {
	KubeClientset    kubernetes.Interface
	DynamicClientset dynamic.Interface
	AppClientset     appclientset.Interface
	AppSetClientset  dynamic.ResourceInterface
}

// GetE2EFixtureK8sClient initializes the Kubernetes clients (if needed), and returns the most recently initalized value.
// Note: this requires a local Kubernetes configuration (for example, while running the E2E tests).
func GetE2EFixtureK8sClient() *E2EFixtureK8sClient {

	// Initialize the Kubernetes clients only on first use
	clientInitialized.Do(func() {

		// set-up variables
		config := getKubeConfig("", clientcmd.ConfigOverrides{})

		internalClientVars = &E2EFixtureK8sClient{
			AppClientset:     appclientset.NewForConfigOrDie(config),
			DynamicClientset: dynamic.NewForConfigOrDie(config),
			KubeClientset:    kubernetes.NewForConfigOrDie(config),
		}

		internalClientVars.AppSetClientset = internalClientVars.DynamicClientset.Resource(v1alpha1.GroupVersion.WithResource("applicationsets")).Namespace(ArgoCDNamespace)

	})
	return internalClientVars
}

// EnsureCleanSlate ensures that the Kubernetes resources on the cluster are are in a 'clean' state, before a test is run.
func EnsureCleanState(t *testing.T) {

	start := time.Now()

	fixtureClient := GetE2EFixtureK8sClient()

	policy := v1.DeletePropagationForeground

	// Delete the applicationset-e2e namespace, if it exists
	err := fixtureClient.KubeClientset.CoreV1().Namespaces().Delete(context.Background(), ApplicationSetNamespace, v1.DeleteOptions{PropagationPolicy: &policy})
	if err != nil && !strings.Contains(err.Error(), "not found") { // 'not found' error is expected
		CheckError(err)
	}

	// delete resources
	// kubectl delete applicationsets --all
	CheckError(fixtureClient.AppSetClientset.DeleteCollection(context.Background(), v1.DeleteOptions{PropagationPolicy: &policy}, v1.ListOptions{}))
	// kubectl delete apps --all
	CheckError(fixtureClient.AppClientset.ArgoprojV1alpha1().Applications(ArgoCDNamespace).DeleteCollection(context.Background(), v1.DeleteOptions{PropagationPolicy: &policy}, v1.ListOptions{}))

	// kubectl delete secrets -l e2e.argoproj.io=true
	CheckError(fixtureClient.KubeClientset.CoreV1().Secrets(ArgoCDNamespace).DeleteCollection(context.Background(),
		v1.DeleteOptions{PropagationPolicy: &policy}, v1.ListOptions{LabelSelector: TestingLabel + "=true"}))

	CheckError(waitForExpectedClusterState())

	// remove tmp dir
	CheckError(os.RemoveAll(TmpDir))

	// create tmp dir
	FailOnErr(Run("", "mkdir", "-p", TmpDir))

	log.WithFields(log.Fields{"duration": time.Since(start), "name": t.Name(), "id": id, "username": "admin", "password": "password"}).Info("clean state")
}

func waitForExpectedClusterState() error {

	fixtureClient := GetE2EFixtureK8sClient()
	// Wait up to 60 seconds for all the ApplicationSets to delete
	if err := waitForSuccess(func() error {
		list, err := fixtureClient.AppSetClientset.List(context.Background(), v1.ListOptions{})
		if err != nil {
			return err
		}
		if list != nil && len(list.Items) > 0 {
			// Fail
			msg := fmt.Sprintf("Waiting for list of ApplicationSets to be size zero: %d", len(list.Items))
			// Intentionally not making this an Errorf, so it can be printf-ed for debugging purposes.
			return errors.New(msg)
		}

		return nil // Pass
	}, time.Now().Add(60*time.Second)); err != nil {
		return err
	}

	// Wait up to 60 seconds for all the Applications to delete
	if err := waitForSuccess(func() error {
		appList, err := fixtureClient.AppClientset.ArgoprojV1alpha1().Applications(ArgoCDNamespace).List(context.Background(), v1.ListOptions{})
		if err != nil {
			return err
		}
		if appList != nil && len(appList.Items) > 0 {
			// Fail
			msg := fmt.Sprintf("Waiting for list of Applications to be size zero: %d", len(appList.Items))
			return errors.New(msg)
		}
		return nil // Pass

	}, time.Now().Add(60*time.Second)); err != nil {
		return err
	}

	// Wait up to 120 seconds for namespace to not exist
	if err := waitForSuccess(func() error {
		_, err := fixtureClient.KubeClientset.CoreV1().Namespaces().Get(context.Background(), ApplicationSetNamespace, v1.GetOptions{})

		msg := ""

		if err == nil {
			msg = fmt.Sprintf("namespace '%s' still exists, after delete", ApplicationSetNamespace)
		}

		if msg == "" && err != nil && strings.Contains(err.Error(), "not found") {
			// Success is an error containing 'applicationset-e2e' not found.
			return nil
		}

		if msg == "" {
			msg = err.Error()
		}

		return errors.New(msg)

	}, time.Now().Add(120*time.Second)); err != nil {
		return err
	}

	return nil
}

// waitForSuccess waits for the condition to return a non-error value.
// Returns if condition returns nil, or the expireTime has elapsed (in which
// case the last error will be returned)
func waitForSuccess(condition func() error, expireTime time.Time) error {

	var mostRecentError error

	for {
		if time.Now().After(expireTime) {
			break
		}

		conditionErr := condition()
		if conditionErr != nil {
			// Fail!
			mostRecentError = conditionErr
		} else {
			// Pass!
			mostRecentError = nil
			break
		}

		// Wait 0.5 seconds on fail
		time.Sleep(500 * time.Millisecond)
	}
	return mostRecentError

}

// getKubeConfig creates new kubernetes client config using specified config path and config overrides variables
func getKubeConfig(configPath string, overrides clientcmd.ConfigOverrides) *rest.Config {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.ExplicitPath = configPath
	clientConfig := clientcmd.NewInteractiveDeferredLoadingClientConfig(loadingRules, &overrides, os.Stdin)

	restConfig, err := clientConfig.ClientConfig()
	CheckError(err)
	return restConfig
}

// creates e2e tests fixture: ensures that Application CRD is installed, creates temporal namespace, starts repo and api server,
// configure currently available cluster.
func init() {

	// ensure we log all shell execs
	log.SetLevel(log.DebugLevel)
}

// PrettyPrintJson is a utility function for debugging purposes
func PrettyPrintJson(obj interface{}) string {
	bytes, err := json.MarshalIndent(obj, "", "    ")
	if err != nil {
		return err.Error()
	}
	return string(bytes)
}

// returns dns friends string which is no longer than 63 characters and has specified postfix at the end
func DnsFriendly(str string, postfix string) string {
	matchFirstCap := regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap := regexp.MustCompile("([a-z0-9])([A-Z])")

	str = matchFirstCap.ReplaceAllString(str, "${1}-${2}")
	str = matchAllCap.ReplaceAllString(str, "${1}-${2}")
	str = strings.ToLower(str)

	if diff := len(str) + len(postfix) - 63; diff > 0 {
		str = str[:len(str)-diff]
	}
	return str + postfix
}

func MustToUnstructured(obj interface{}) *unstructured.Unstructured {
	uObj, err := ToUnstructured(obj)
	if err != nil {
		panic(err)
	}
	return uObj
}

// ToUnstructured converts a concrete K8s API type to an unstructured object
func ToUnstructured(obj interface{}) (*unstructured.Unstructured, error) {
	uObj, err := runtime.NewTestUnstructuredConverter(equality.Semantic).ToUnstructured(obj)
	if err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: uObj}, nil
}
