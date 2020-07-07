package controllers

import (
	"context"
	argov1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCreateApplications(t *testing.T) {

	scheme := runtime.NewScheme()
	argov1alpha1.AddToScheme(scheme)

	c := fake.NewFakeClientWithScheme(scheme)

	r := ApplicationSetReconciler{
		Client: c,
		Scheme: scheme,
		Recorder: record.NewFakeRecorder(1),
	}

	appSet := argoprojiov1alpha1.ApplicationSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:                       "name",
			Namespace:                  "namespace",
		},
	}

	apps := []argov1alpha1.Application{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:                       "app1",
			},
		},
	}

	r.createApplications(context.TODO(), appSet, apps)


	got := &argov1alpha1.Application{}
	_ = c.Get(context.Background(), client.ObjectKey{
		Namespace: "namespace",
		Name:      "app1",
	}, got)


	expected := argov1alpha1.Application{
		TypeMeta:   metav1.TypeMeta{
			Kind:       "Application",
			APIVersion: "argoproj.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:                       "app1",
			Namespace:                  "namespace",
			ResourceVersion:			"1",
		},
	}

	assert.Equal(t, expected, *got)

}