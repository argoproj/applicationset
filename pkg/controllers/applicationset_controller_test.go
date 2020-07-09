package controllers

import (
	"context"
	argov1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	crtclient "sigs.k8s.io/controller-runtime/pkg/client"
	"testing"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCreateApplications(t *testing.T) {

	scheme := runtime.NewScheme()
	argov1alpha1.AddToScheme(scheme)

	client := fake.NewFakeClientWithScheme(scheme)

	r := ApplicationSetReconciler{
		Client:   client,
		Scheme:   scheme,
		Recorder: record.NewFakeRecorder(1),
	}

	for _, c := range []struct {
		appSet   argoprojiov1alpha1.ApplicationSet
		apps     []argov1alpha1.Application
		expected []argov1alpha1.Application
	}{
		{
			appSet: argoprojiov1alpha1.ApplicationSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "namespace",
				},
			},
			apps: []argov1alpha1.Application{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "app1",
					},
				},
			},
			expected: []argov1alpha1.Application{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Application",
						APIVersion: "argoproj.io/v1alpha1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:            "app1",
						Namespace:       "namespace",
						ResourceVersion: "1",
					},
				},
			},
		},
	} {
		r.createApplications(context.TODO(), c.appSet, c.apps)

		for _, e := range c.expected {
			got := &argov1alpha1.Application{}
			_ = client.Get(context.Background(), crtclient.ObjectKey{
				Namespace: e.Namespace,
				Name:      e.Name,
			}, got)

			assert.Equal(t, e, *got)
		}
	}

}
