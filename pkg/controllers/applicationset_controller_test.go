package controllers

import (
	"context"
	"fmt"
	argov1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	crtclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"testing"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
)

func TestCreateOrUpdateApplications(t *testing.T) {

	scheme := runtime.NewScheme()
	argoprojiov1alpha1.AddToScheme(scheme)
	argov1alpha1.AddToScheme(scheme)

	for _, c := range []struct {
		appSet     argoprojiov1alpha1.ApplicationSet
		existsApps []argov1alpha1.Application
		apps       []argov1alpha1.Application
		expected   []argov1alpha1.Application
	}{
		{
			appSet: argoprojiov1alpha1.ApplicationSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "namespace",
				},
			},
			existsApps: nil,
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
		{
			appSet: argoprojiov1alpha1.ApplicationSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "namespace",
				},
				Spec: argoprojiov1alpha1.ApplicationSetSpec{
					Template: argoprojiov1alpha1.ApplicationSetTemplate{
						Spec: argov1alpha1.ApplicationSpec{
							Project: "project",
						},
					},
				},
			},
			existsApps: []argov1alpha1.Application{
				argov1alpha1.Application{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Application",
						APIVersion: "argoproj.io/v1alpha1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:            "app1",
						Namespace:       "namespace",
						ResourceVersion: "2",
					},
					Spec: argov1alpha1.ApplicationSpec{
						Project: "test",
					},
				},
			},
			apps: []argov1alpha1.Application{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "app1",
					},
					Spec: argov1alpha1.ApplicationSpec{
						Project: "project",
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
						ResourceVersion: "3",
					},
					Spec: argov1alpha1.ApplicationSpec{
						Project: "project",
					},
				},
			},
		},
		{
			appSet: argoprojiov1alpha1.ApplicationSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "namespace",
				},
				Spec: argoprojiov1alpha1.ApplicationSetSpec{
					Template: argoprojiov1alpha1.ApplicationSetTemplate{
						Spec: argov1alpha1.ApplicationSpec{
							Project: "project",
						},
					},
				},
			},
			existsApps: []argov1alpha1.Application{
				argov1alpha1.Application{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Application",
						APIVersion: "argoproj.io/v1alpha1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:            "app1",
						Namespace:       "namespace",
						ResourceVersion: "2",
					},
					Spec: argov1alpha1.ApplicationSpec{
						Project: "test",
					},
				},
			},
			apps: []argov1alpha1.Application{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "app2",
					},
					Spec: argov1alpha1.ApplicationSpec{
						Project: "project",
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
						Name:            "app2",
						Namespace:       "namespace",
						ResourceVersion: "1",
					},
					Spec: argov1alpha1.ApplicationSpec{
						Project: "project",
					},
				},
			},
		},
	} {
		initObjs := []runtime.Object{&c.appSet}
		for _, a := range c.existsApps {
			controllerutil.SetControllerReference(&c.appSet, &a, scheme)
			initObjs = append(initObjs, &a)
		}

		client := fake.NewFakeClientWithScheme(scheme, initObjs...)

		r := ApplicationSetReconciler{
			Client:   client,
			Scheme:   scheme,
			Recorder: record.NewFakeRecorder(len(initObjs) + len(c.expected)),
		}

		r.createOrUpdateInCluster(context.TODO(), c.appSet, c.apps)

		for _, obj := range c.expected {
			got := &argov1alpha1.Application{}
			_ = client.Get(context.Background(), crtclient.ObjectKey{
				Namespace: obj.Namespace,
				Name:      obj.Name,
			}, got)

			controllerutil.SetControllerReference(&c.appSet, &obj, r.Scheme)

			assert.Equal(t, obj, *got)
		}
	}

}

func TestDeleteApplications(t *testing.T) {

	scheme := runtime.NewScheme()
	argoprojiov1alpha1.AddToScheme(scheme)
	argov1alpha1.AddToScheme(scheme)

	for _, c := range []struct {
		appSet      argoprojiov1alpha1.ApplicationSet
		existsApps  []argov1alpha1.Application
		apps        []argov1alpha1.Application
		expected    []argov1alpha1.Application
		notExpected []argov1alpha1.Application
	}{
		{
			appSet: argoprojiov1alpha1.ApplicationSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "namespace",
				},
				Spec: argoprojiov1alpha1.ApplicationSetSpec{
					Template: argoprojiov1alpha1.ApplicationSetTemplate{
						Spec: argov1alpha1.ApplicationSpec{
							Project: "project",
						},
					},
				},
			},
			existsApps: []argov1alpha1.Application{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Application",
						APIVersion: "argoproj.io/v1alpha1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:            "delete",
						Namespace:       "namespace",
						ResourceVersion: "2",
					},
					Spec: argov1alpha1.ApplicationSpec{
						Project: "project",
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Application",
						APIVersion: "argoproj.io/v1alpha1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:            "keep",
						Namespace:       "namespace",
						ResourceVersion: "2",
					},
					Spec: argov1alpha1.ApplicationSpec{
						Project: "project",
					},
				},
			},
			apps: []argov1alpha1.Application{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "keep",
					},
					Spec: argov1alpha1.ApplicationSpec{
						Project: "project",
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
						Name:            "keep",
						Namespace:       "namespace",
						ResourceVersion: "2",
					},
					Spec: argov1alpha1.ApplicationSpec{
						Project: "project",
					},
				},
			},
			notExpected: []argov1alpha1.Application{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Application",
						APIVersion: "argoproj.io/v1alpha1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:            "delete",
						Namespace:       "namespace",
						ResourceVersion: "1",
					},
					Spec: argov1alpha1.ApplicationSpec{
						Project: "project",
					},
				},
			},
		},
	} {
		initObjs := []runtime.Object{&c.appSet}
		for _, a := range c.existsApps {
			temp := a
			controllerutil.SetControllerReference(&c.appSet, &temp, scheme)
			initObjs = append(initObjs, &temp)
		}

		client := fake.NewFakeClientWithScheme(scheme, initObjs...)

		r := ApplicationSetReconciler{
			Client:   client,
			Scheme:   scheme,
			Recorder: record.NewFakeRecorder(len(initObjs) + len(c.expected)),
		}

		r.deleteInCluster(context.TODO(), c.appSet, c.apps)

		for _, obj := range c.expected {
			got := &argov1alpha1.Application{}
			_ = client.Get(context.Background(), crtclient.ObjectKey{
				Namespace: obj.Namespace,
				Name:      obj.Name,
			}, got)

			controllerutil.SetControllerReference(&c.appSet, &obj, r.Scheme)

			assert.Equal(t, obj, *got)
		}

		for _, obj := range c.notExpected {
			got := &argov1alpha1.Application{}
			err := client.Get(context.Background(), crtclient.ObjectKey{
				Namespace: obj.Namespace,
				Name:      obj.Name,
			}, got)

			assert.EqualError(t, err, fmt.Sprintf("applications.argoproj.io \"%s\" not found", obj.Name))
		}
	}

}
