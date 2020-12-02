package utils

import (
	"context"
	"fmt"
	argov1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// override "sigs.k8s.io/controller-runtime" CreateOrUpdate function to add equity function for argov1alpha1.ApplicationDestination
// argov1alpha1.ApplicationDestination has a private variable, so the default implementation fails to compare it
func CreateOrUpdate(ctx context.Context, c client.Client, obj runtime.Object, f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
	key, err := client.ObjectKeyFromObject(obj)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	if err := c.Get(ctx, key, obj); err != nil {
		if !errors.IsNotFound(err) {
			return controllerutil.OperationResultNone, err
		}
		if err := mutate(f, key, obj); err != nil {
			return controllerutil.OperationResultNone, err
		}
		if err := c.Create(ctx, obj); err != nil {
			return controllerutil.OperationResultNone, err
		}
		return controllerutil.OperationResultCreated, nil
	}

	existing := obj.DeepCopyObject()
	if err := mutate(f, key, obj); err != nil {
		return controllerutil.OperationResultNone, err
	}

	equality := conversion.EqualitiesOrDie(
		func(a, b resource.Quantity) bool {
			// Ignore formatting, only care that numeric value stayed the same.
			// TODO: if we decide it's important, it should be safe to start comparing the format.
			//
			// Uninitialized quantities are equivalent to 0 quantities.
			return a.Cmp(b) == 0
		},
		func(a, b metav1.MicroTime) bool {
			return a.UTC() == b.UTC()
		},
		func(a, b metav1.Time) bool {
			return a.UTC() == b.UTC()
		},
		func(a, b labels.Selector) bool {
			return a.String() == b.String()
		},
		func(a, b fields.Selector) bool {
			return a.String() == b.String()
		},
		func(a, b argov1alpha1.ApplicationDestination) bool {
			return a.Namespace == b.Namespace && a.Name == b.Name && a.Server == b.Server
		},
	)

	if equality.DeepEqual(existing, obj) {
		return controllerutil.OperationResultNone, nil
	}

	if err := c.Update(ctx, obj); err != nil {
		return controllerutil.OperationResultNone, err
	}
	return controllerutil.OperationResultUpdated, nil
}

// mutate wraps a MutateFn and applies validation to its result
func mutate(f controllerutil.MutateFn, key client.ObjectKey, obj runtime.Object) error {
	if err := f(); err != nil {
		return err
	}
	if newKey, err := client.ObjectKeyFromObject(obj); err != nil || key != newKey {
		return fmt.Errorf("MutateFn cannot mutate object name and/or object namespace")
	}
	return nil
}
