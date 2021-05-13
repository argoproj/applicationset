package generators

import (
	"context"
	"errors"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	dynfake "k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"testing"
)

const resourceApiVersion = "mallard.io/v1"
const resourceKind = "ducks"
const resourceName = "quak"

func TestGenerateParamsForDuckType(t *testing.T) {
	clusters := []client.Object{
		&corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "staging-01",
				Namespace: "namespace",
				Labels: map[string]string{
					"argocd.argoproj.io/secret-type": "cluster",
					"environment":                    "staging",
					"org":                            "foo",
				},
				Annotations: map[string]string{
					"foo.argoproj.io": "staging",
				},
			},
			Data: map[string][]byte{
				"config": []byte("{}"),
				"name":   []byte("staging-01"),
				"server": []byte("https://staging-01.example.com"),
			},
			Type: corev1.SecretType("Opaque"),
		},
		&corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "production-01",
				Namespace: "namespace",
				Labels: map[string]string{
					"argocd.argoproj.io/secret-type": "cluster",
					"environment":                    "production",
					"org":                            "bar",
				},
				Annotations: map[string]string{
					"foo.argoproj.io": "production",
				},
			},
			Data: map[string][]byte{
				"config": []byte("{}"),
				"name":   []byte("production-01"),
				"server": []byte("https://production-01.example.com"),
			},
			Type: corev1.SecretType("Opaque"),
		},
	}

	duckType := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": resourceApiVersion,
			"kind":       "Duck",
			"metadata": map[string]interface{}{
				"name":      resourceName,
				"namespace": "namespace",
			},
			"status": map[string]interface{}{
				"decisions": []interface{}{
					map[string]interface{}{
						"clusterName": "staging-01",
					},
					map[string]interface{}{
						"clusterName": "production-01",
					},
				},
			},
		},
	}

	duckTypeProdOnly := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": resourceApiVersion,
			"kind":       "Duck",
			"metadata": map[string]interface{}{
				"name":      resourceName,
				"namespace": "namespace",
			},
			"status": map[string]interface{}{
				"decisions": []interface{}{
					map[string]interface{}{
						"clusterName": "production-01",
					},
				},
			},
		},
	}

	duckTypeEmpty := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": resourceApiVersion,
			"kind":       "Duck",
			"metadata": map[string]interface{}{
				"name":      resourceName,
				"namespace": "namespace",
			},
			"status": map[string]interface{}{},
		},
	}

	testCases := []struct {
		name         string
		apiVersion   string
		kind         string
		resourceName string
		resource     *unstructured.Unstructured
		values       map[string]string
		expected     []map[string]string
		// clientError is true if a k8s client error should be simulated
		clientError   bool
		expectedError error
	}{
		{
			name:          "no duck resource",
			apiVersion:    "",
			kind:          "",
			resourceName:  "",
			resource:      duckType,
			values:        nil,
			expected:      []map[string]string{},
			clientError:   false,
			expectedError: errors.New("Invalid resource reference"),
		},
		{
			name:          "invalid params for duck resource",
			apiVersion:    resourceApiVersion,
			kind:          "badvalue",
			resourceName:  resourceName,
			resource:      duckType,
			values:        nil,
			expected:      []map[string]string{},
			clientError:   false,
			expectedError: errors.New("duck.mallard.io \"quak\" not found"),
		},
		{
			name:         "duck type generator",
			apiVersion:   resourceApiVersion,
			kind:         resourceKind,
			resourceName: resourceName,
			resource:     duckType,
			values:       nil,
			expected: []map[string]string{
				{"name": "production-01", "server": "https://production-01.example.com"},

				{"name": "staging-01", "server": "https://staging-01.example.com"},
			},
			clientError:   false,
			expectedError: nil,
		},
		{
			name:         "production-only",
			apiVersion:   resourceApiVersion,
			kind:         resourceKind,
			resourceName: resourceName,
			resource:     duckTypeProdOnly,
			values: map[string]string{
				"foo": "bar",
			},
			expected: []map[string]string{
				{"values.foo": "bar", "name": "production-01", "server": "https://production-01.example.com"},
			},
			clientError:   false,
			expectedError: nil,
		},
		{
			name:          "duck type empty status",
			apiVersion:    resourceApiVersion,
			kind:          resourceKind,
			resourceName:  resourceName,
			resource:      duckTypeEmpty,
			values:        nil,
			expected:      nil,
			clientError:   false,
			expectedError: nil,
		},
		{
			name:          "simulate client error",
			apiVersion:    resourceApiVersion,
			kind:          resourceKind,
			resourceName:  resourceName,
			resource:      duckType,
			values:        nil,
			expected:      nil,
			clientError:   true,
			expectedError: errors.New("could not list Secrets"),
		},
	}

	// convert []client.Object to []runtime.Object, for use by kubefake package
	runtimeClusters := []runtime.Object{}
	for _, clientCluster := range clusters {
		runtimeClusters = append(runtimeClusters, clientCluster)
	}

	for _, testCase := range testCases {

		t.Run(testCase.name, func(t *testing.T) {

			appClientset := kubefake.NewSimpleClientset(runtimeClusters...)

			fakeDynClient := dynfake.NewSimpleDynamicClient(runtime.NewScheme(), testCase.resource)

			var duckTypeGenerator = NewDuckTypeGenerator(context.Background(), fakeDynClient, appClientset, "namespace")

			got, err := duckTypeGenerator.GenerateParams(&argoprojiov1alpha1.ApplicationSetGenerator{
				DuckType: &argoprojiov1alpha1.DuckTypeGenerator{
					ApiVersion: testCase.apiVersion,
					Kind:       testCase.kind,
					Name:       testCase.resourceName,
					Values:     testCase.values,
				},
			}, nil)

			if testCase.expectedError != nil {
				assert.Error(t, testCase.expectedError, err)
			} else {
				assert.NoError(t, err)
				assert.ElementsMatch(t, testCase.expected, got)
			}

		})
	}
}
