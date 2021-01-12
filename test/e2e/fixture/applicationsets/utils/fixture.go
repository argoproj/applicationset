package utils

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/argoproj-labs/applicationset/api/v1alpha1"
	log "github.com/sirupsen/logrus"

	appclientset "github.com/argoproj/argo-cd/pkg/client/clientset/versioned"
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
	adminPassword   = "password"
	ArgoCDNamespace = "argocd-e2e"
	TmpDir          = "/tmp/applicationset-e2e"
	TestingLabel    = "e2e.argoproj.io"
)

var (
	id               string
	name             string
	KubeClientset    kubernetes.Interface
	DynamicClientset dynamic.Interface
	AppClientset     appclientset.Interface
	AppSetClientset  dynamic.ResourceInterface

	token     string
	plainText bool
)

func EnsureCleanState(t *testing.T) {

	start := time.Now()

	policy := v1.DeletePropagationBackground
	// delete resources
	// kubectl delete applicationsets --all
	CheckError(AppSetClientset.DeleteCollection(context.Background(), v1.DeleteOptions{PropagationPolicy: &policy}, v1.ListOptions{}))
	// kubectl delete apps --all
	CheckError(AppClientset.ArgoprojV1alpha1().Applications(ArgoCDNamespace).DeleteCollection(context.Background(), v1.DeleteOptions{PropagationPolicy: &policy}, v1.ListOptions{}))

	// kubectl delete secrets -l e2e.argoproj.io=true
	CheckError(KubeClientset.CoreV1().Secrets(ArgoCDNamespace).DeleteCollection(context.Background(),
		v1.DeleteOptions{PropagationPolicy: &policy}, v1.ListOptions{LabelSelector: TestingLabel + "=true"}))

	// remove tmp dir
	CheckError(os.RemoveAll(TmpDir))

	// create tmp dir
	FailOnErr(Run("", "mkdir", "-p", TmpDir))

	log.WithFields(log.Fields{"duration": time.Since(start), "name": t.Name(), "id": id, "username": "admin", "password": "password"}).Info("clean state")
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

	// set-up variables
	config := getKubeConfig("", clientcmd.ConfigOverrides{})
	AppClientset = appclientset.NewForConfigOrDie(config)
	KubeClientset = kubernetes.NewForConfigOrDie(config)
	DynamicClientset = dynamic.NewForConfigOrDie(config)
	AppSetClientset = DynamicClientset.Resource(v1alpha1.GroupVersion.WithResource("applicationsets")).Namespace(ArgoCDNamespace)

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
