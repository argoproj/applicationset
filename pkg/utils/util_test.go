package utils

import (
	"testing"

	argov1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestRenderTemplateParams(t *testing.T) {

	// Believe it or not, this is actually less complex than the equivalent solution using reflection
	fieldMap := map[string]func(app *argov1alpha1.Application) *string{}
	fieldMap["Path"] = func(app *argov1alpha1.Application) *string { return &app.Spec.Source.Path }
	fieldMap["RepoURL"] = func(app *argov1alpha1.Application) *string { return &app.Spec.Source.RepoURL }
	fieldMap["TargetRevision"] = func(app *argov1alpha1.Application) *string { return &app.Spec.Source.TargetRevision }
	fieldMap["Chart"] = func(app *argov1alpha1.Application) *string { return &app.Spec.Source.Chart }

	fieldMap["Server"] = func(app *argov1alpha1.Application) *string { return &app.Spec.Destination.Server }
	fieldMap["Namespace"] = func(app *argov1alpha1.Application) *string { return &app.Spec.Destination.Namespace }
	fieldMap["Name"] = func(app *argov1alpha1.Application) *string { return &app.Spec.Destination.Name }

	fieldMap["Project"] = func(app *argov1alpha1.Application) *string { return &app.Spec.Project }

	emptyApplication := &argov1alpha1.Application{
		Spec: argov1alpha1.ApplicationSpec{
			Source: argov1alpha1.ApplicationSource{
				Path:           "",
				RepoURL:        "",
				TargetRevision: "",
				Chart:          "",
			},
			Destination: argov1alpha1.ApplicationDestination{
				Server:    "",
				Namespace: "",
				Name:      "",
			},
			Project: "",
		},
	}

	tests := []struct {
		name        string
		fieldVal    string
		params      map[string]string
		expectedVal string
	}{
		{
			name:        "simple substitution",
			fieldVal:    "{{one}}",
			expectedVal: "two",
			params: map[string]string{
				"one": "two",
			},
		},
		{
			name:        "simple substitution with whitespace",
			fieldVal:    "{{ one }}",
			expectedVal: "two",
			params: map[string]string{
				"one": "two",
			},
		},

		{
			name:        "template characters but not in a template",
			fieldVal:    "}} {{",
			expectedVal: "}} {{",
			params: map[string]string{
				"one": "two",
			},
		},

		{
			name:        "nested template",
			fieldVal:    "{{ }}",
			expectedVal: "{{ }}",
			params: map[string]string{
				"one": "{{ }}",
			},
		},
		{
			name:        "field with whitespace",
			fieldVal:    "{{ }}",
			expectedVal: "{{ }}",
			params: map[string]string{
				" ": "two",
				"":  "three",
			},
		},

		{
			name:        "template contains itself, containing itself",
			fieldVal:    "{{one}}",
			expectedVal: "{{one}}",
			params: map[string]string{
				"{{one}}": "{{one}}",
			},
		},

		{
			name:        "template contains itself, containing something else",
			fieldVal:    "{{one}}",
			expectedVal: "{{one}}",
			params: map[string]string{
				"{{one}}": "{{two}}",
			},
		},

		{
			name:        "templates are case sensitive",
			fieldVal:    "{{ONE}}",
			expectedVal: "{{ONE}}",
			params: map[string]string{
				"{{one}}": "two",
			},
		},
		{
			name:        "multiple on a line",
			fieldVal:    "{{one}}{{one}}",
			expectedVal: "twotwo",
			params: map[string]string{
				"one": "two",
			},
		},
		{
			name:        "multiple different on a line",
			fieldVal:    "{{one}}{{three}}",
			expectedVal: "twofour",
			params: map[string]string{
				"one":   "two",
				"three": "four",
			},
		},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			for fieldName, getPtrFunc := range fieldMap {

				// Clone the template application
				application := emptyApplication.DeepCopy()

				// Set the value of the target field, to the test value
				*getPtrFunc(application) = test.fieldVal

				// Render the cloned application, into a new application
				render := Render{}
				newApplication, err := render.RenderTemplateParams(application, nil, test.params)

				// Retrieve the value of the target field from the newApplication, then verify that
				// the target field has been templated into the expected value
				actualValue := *getPtrFunc(newApplication)
				assert.Equal(t, test.expectedVal, actualValue, "Field '%s' had an unexpected value. expected: '%s' value: '%s'", fieldName, test.expectedVal, actualValue)
				assert.NoError(t, err)

			}
		})
	}

}
