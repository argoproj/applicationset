package generators

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	argov1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"

	"testing"

	"github.com/stretchr/testify/assert"
)

type possiblyErroringFakeCtrlRuntimeClient struct {
	client.Client
	shouldError bool
}

func (p *possiblyErroringFakeCtrlRuntimeClient) List(ctx context.Context, secretList runtime.Object, opts ...client.ListOption) error {
	if p.shouldError {
		return errors.New("could not list Secrets")
	}
	return p.Client.List(ctx, secretList, opts...)
}

func getRenderTemplate(appName, clusterName, server, environmentLabel string) *argov1alpha1.Application {
	return &argov1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", clusterName, appName),
			Namespace: "namespace",
			Finalizers: []string{
				"resources-finalizer.argocd.argoproj.io",
			},
			Labels: map[string]string{
				"environment": environmentLabel,
			},
		},
		Spec: argov1alpha1.ApplicationSpec{
			Source: argov1alpha1.ApplicationSource{
				RepoURL:        "RepoURL",
				Path:           fmt.Sprintf("%s/%s", appName, environmentLabel),
				TargetRevision: "HEAD",
			},
			Destination: argov1alpha1.ApplicationDestination{
				Server:    server,
				Namespace: "destinationNamespace",
			},
			Project: "project",
		},
	}
}

func TestGenerateApplications(t *testing.T) {
	clusters := []runtime.Object{
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
			},
			Data: map[string][]byte{
				"config": []byte(base64.StdEncoding.EncodeToString([]byte("foo"))),
				"name":   []byte(base64.StdEncoding.EncodeToString([]byte("staging-01"))),
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
			},
			Data: map[string][]byte{
				"config": []byte(base64.StdEncoding.EncodeToString([]byte("foo"))),
				"name":   []byte(base64.StdEncoding.EncodeToString([]byte("production-01"))),
				"server": []byte("https://production-01.example.com"),
			},
			Type: corev1.SecretType("Opaque"),
		},
	}

	applicationSetTemplate := argoprojiov1alpha1.ApplicationSetTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"environment": "{{metadata.labels.environment}}",
			},
			Name:      "{{name}}-app",
			Namespace: "namespace",
		},
		Spec: argov1alpha1.ApplicationSpec{
			Source: argov1alpha1.ApplicationSource{
				RepoURL:        "RepoURL",
				Path:           "app/{{metadata.labels.environment}}",
				TargetRevision: "HEAD",
			},
			Destination: argov1alpha1.ApplicationDestination{
				Server:    "{{server}}",
				Namespace: "destinationNamespace",
			},
			Project: "project",
		},
	}

	testCases := []struct {
		template      argoprojiov1alpha1.ApplicationSetTemplate
		selector      metav1.LabelSelector
		expected      []argov1alpha1.Application
		clientError   bool
		expectedError error
	}{
		{
			applicationSetTemplate,
			metav1.LabelSelector{},
			[]argov1alpha1.Application{
				*getRenderTemplate("app", "staging-01", "https://staging-01.example.com", "staging"),
				*getRenderTemplate("app", "production-01", "https://production-01.example.com", "production"),
			},
			false,
			nil,
		},
		{
			applicationSetTemplate,
			metav1.LabelSelector{
				MatchLabels: map[string]string{
					"environment": "production",
				},
			},
			[]argov1alpha1.Application{
				*getRenderTemplate("app", "production-01", "https://production-01.example.com", "production"),
			},
			false,
			nil,
		},
		{
			applicationSetTemplate,
			metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "environment",
						Operator: "In",
						Values: []string{
							"production",
							"staging",
						},
					},
				},
			},
			[]argov1alpha1.Application{
				*getRenderTemplate("app", "staging-01", "https://staging-01.example.com", "staging"),
				*getRenderTemplate("app", "production-01", "https://production-01.example.com", "production"),
			},
			false,
			nil,
		},
		{
			applicationSetTemplate,
			metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "environment",
						Operator: "In",
						Values: []string{
							"production",
							"staging",
						},
					},
				},
				MatchLabels: map[string]string{
					"org": "foo",
				},
			},
			[]argov1alpha1.Application{
				*getRenderTemplate("app", "staging-01", "https://staging-01.example.com", "staging"),
			},
			false,
			nil,
		},
		{
			applicationSetTemplate,
			metav1.LabelSelector{},
			nil,
			true,
			errors.New("could not list Secrets"),
		},
	}

	for _, testCase := range testCases {
		fakeClient := fake.NewFakeClientWithScheme(scheme.Scheme, clusters...)
		cl := &possiblyErroringFakeCtrlRuntimeClient{
			fakeClient,
			testCase.clientError,
		}

		var clusterGenerator = NewClusterGenerator(cl)
		applicationSetInfo := argoprojiov1alpha1.ApplicationSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "set",
			},
			Spec: argoprojiov1alpha1.ApplicationSetSpec{
				Generators: []argoprojiov1alpha1.ApplicationSetGenerator{{
					Clusters: &argoprojiov1alpha1.ClusterGenerator{
						Selector: testCase.selector,
					},
				}},
				Template: testCase.template,
			},
		}

		got, err := clusterGenerator.GenerateApplications(&applicationSetInfo.Spec.Generators[0], &applicationSetInfo)

		if testCase.expectedError != nil {
			assert.Error(t, testCase.expectedError, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, testCase.expected, got)
		}

	}
}
