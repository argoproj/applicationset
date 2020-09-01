package generators

import (
	"context"

	"encoding/base64"
	"errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"testing"
)

type possiblyErroringFakeCtrlRuntimeClient struct {
	client.Client
	shouldError bool
}

func (p *possiblyErroringFakeCtrlRuntimeClient) List(ctx context.Context, secretList runtime.Object, opts ...client.ListOption) error {
	if p.shouldError {
		return errors.New("Could not list Secrets.")
	}
	return p.Client.List(ctx, secretList, opts...)
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
				},
			},
			Data: map[string][]byte{
				"config": []byte(base64.StdEncoding.EncodeToString([]byte("foo"))),
				"name":   []byte(base64.StdEncoding.EncodeToString([]byte("productuion-01"))),
				"server": []byte("https://production-01.example.com"),
			},
			Type: corev1.SecretType("Opaque"),
		},
	}

	testCases := []struct {
		selector      metav1.LabelSelector
		expected      []map[string]string
		clientError   bool
		expectedError error
	}{
		{
			metav1.LabelSelector{},
			[]map[string]string{
				{"name": "staging-01", "server": "https://staging-01.example.com", "metadata.labels.environment": "staging", "metadata.labels.argocd.argoproj.io/secret-type": "cluster"},
				{"name": "production-01", "server": "https://production-01.example.com", "metadata.labels.environment": "production", "metadata.labels.argocd.argoproj.io/secret-type": "cluster"},
			},
			false,
			nil,
		},
		{
			metav1.LabelSelector{
				MatchLabels: map[string]string{
					"environment": "production",
				},
			},
			[]map[string]string{
				{"name": "production-01", "server": "https://production-01.example.com", "metadata.labels.environment": "production", "metadata.labels.argocd.argoproj.io/secret-type": "cluster"},
			},
			false,
			nil,
		},
		{
			metav1.LabelSelector{},
			nil,
			true,
			errors.New("Could not list Secrets."),
		},
	}

	for _, testCase := range testCases {
		fakeClient := fake.NewFakeClientWithScheme(scheme.Scheme, clusters...)
		cl := &possiblyErroringFakeCtrlRuntimeClient{
			fakeClient,
			testCase.clientError,
		}

		var clusterGenerator = NewClusterGenerator(cl)

		got, err := clusterGenerator.GenerateParams(&argoprojiov1alpha1.ApplicationSetGenerator{
			Clusters: &argoprojiov1alpha1.ClusterGenerator{
				Selector: testCase.selector,
			},
		})

		if testCase.expectedError != nil {
			assert.Error(t, testCase.expectedError, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, testCase.expected, got)
		}

	}
}
