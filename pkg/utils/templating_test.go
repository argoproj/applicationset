package utils

import (
	"testing"

	argoprojiov1alpha1 "github.com/argoproj/applicationset/api/v1alpha1"
	argov1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
	// "sigs.k8s.io/yaml"
)

func composeApplicationFieldsFuncMap() map[string]func(app *argov1alpha1.Application) *string {
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

	return fieldMap
}

func TestRenderTemplateParamsWithFasttemplate(t *testing.T) {

	fieldMap := composeApplicationFieldsFuncMap()

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
		{
			name:        "gotemplate expressions are supported",
			fieldVal:    "${{.one}} ${{.three}}",
			expectedVal: "two four",
			params: map[string]string{
				"one":   "two",
				"three": "four",
			},
		},
		{
			name:        "sprig functions are avilable",
			fieldVal:    `${{upper .one}} ${{default "four" .three}}`,
			expectedVal: "TWO four",
			params: map[string]string{
				"one": "two",
			},
		},
		{
			name:        "mix of gotemplate expressions and fasttemplate",
			fieldVal:    `{{one}} ${{default "four" .three}}`,
			expectedVal: "two four",
			params: map[string]string{
				"one": "two",
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
				newApplication, err := render.RenderTemplateParams(application, nil, nil, test.params)

				// Retrieve the value of the target field from the newApplication, then verify that
				// the target field has been templated into the expected value
				actualValue := *getPtrFunc(newApplication)
				assert.Equal(t, test.expectedVal, actualValue, "Field '%s' had an unexpected value. expected: '%s' value: '%s'", fieldName, test.expectedVal, actualValue)
				assert.NoError(t, err)

			}
		})
	}

}

func TestRenderTemplateParamsWithGotemplate(t *testing.T) {

	tests := []struct {
		name        string
		input       argoprojiov1alpha1.ApplicationSetUntypedTemplate
		params      map[string]string
		expectedApp argov1alpha1.Application
	}{
		{
			name: "advanced gotemplate with sprig",
			input: `
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: 'app.{{ default .cluster .DoesNotExist }}'
  labels:
  {{ range $key, $val := . }}
      {{ if (hasPrefix "appLables." $key ) }}
      {{ $key | replace "appLables." "" }}: {{ $val }}
      {{ end }}
    {{ end }}
spec:
  project: default
  revisionHistoryLimit: {{ atoi .revisionHistoryLimit }}
  source:
    repoURL: https://github.com/argoproj-labs/applicationset.git
    targetRevision: HEAD
    path: examples/list-generator/guestbook/engineering-{{ .cluster }}
    {{ toYaml (fromJson .sourceRendererConf) | nindent 4 }}
  destination:
    server: https://kubernetes.default.svc
    namespace: {{ if (eq .cluster "dev") }}dev-namespace{{ else }}other-namespace{{ end }}
  syncPolicy:
    automated:
      selfHeal: {{ eq .autosyncSelfHeal "true" }}`,

			params: map[string]string{
				"cluster":              "dev",
				"appLables.label1":     "label1",
				"appLables.label2":     "label2",
				"revisionHistoryLimit": "0",
				"sourceRendererConf":   "{\"plugin\": {\"name\": \"kustomized-helm\"} }",
				"autosyncSelfHeal":     "true",
			},
			expectedApp: argov1alpha1.Application{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Application",
					APIVersion: "argoproj.io/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:       "app.dev",
					Finalizers: []string{"resources-finalizer.argocd.argoproj.io"},
					Labels: map[string]string{
						"label1": "label1",
						"label2": "label2",
					},
				},
				Spec: argov1alpha1.ApplicationSpec{
					Source: argov1alpha1.ApplicationSource{
						Path:           "examples/list-generator/guestbook/engineering-dev",
						RepoURL:        "https://github.com/argoproj-labs/applicationset.git",
						TargetRevision: "HEAD",
						Chart:          "",
						Plugin: &argov1alpha1.ApplicationSourcePlugin{
							Name: "kustomized-helm",
						},
					},
					Destination: argov1alpha1.ApplicationDestination{
						Server:    "https://kubernetes.default.svc",
						Namespace: "dev-namespace",
						Name:      "",
					},
					SyncPolicy: &argov1alpha1.SyncPolicy{
						Automated: &argov1alpha1.SyncPolicyAutomated{
							SelfHeal: true,
						},
					},
					Project:              "default",
					RevisionHistoryLimit: new(int64),
				},
			},
		},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			render := Render{}

			newApplication, err := render.RenderTemplateParams(&argov1alpha1.Application{}, &test.input, nil, test.params)

			assert.NoError(t, err)
			assert.Equal(t, *newApplication, test.expectedApp, "Expected: '%s' value: '%s'", *newApplication, test.expectedApp)
		})
	}

}

func TestRenderTemplateParamsFinalizers(t *testing.T) {

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

	for _, c := range []struct {
		testName           string
		syncPolicy         *argoprojiov1alpha1.ApplicationSetSyncPolicy
		existingFinalizers []string
		expectedFinalizers []string
	}{
		{
			testName:           "existing finalizer should be preserved",
			existingFinalizers: []string{"existing-finalizer"},
			syncPolicy:         nil,
			expectedFinalizers: []string{"existing-finalizer"},
		},
		{
			testName:           "background finalizer should be preserved",
			existingFinalizers: []string{"resources-finalizer.argocd.argoproj.io/background"},
			syncPolicy:         nil,
			expectedFinalizers: []string{"resources-finalizer.argocd.argoproj.io/background"},
		},

		{
			testName:           "empty finalizer and empty sync should use standard finalizer",
			existingFinalizers: nil,
			syncPolicy:         nil,
			expectedFinalizers: []string{"resources-finalizer.argocd.argoproj.io"},
		},

		{
			testName:           "standard finalizer should be preserved",
			existingFinalizers: []string{"resources-finalizer.argocd.argoproj.io"},
			syncPolicy:         nil,
			expectedFinalizers: []string{"resources-finalizer.argocd.argoproj.io"},
		},
		{
			testName:           "empty array finalizers should use standard finalizer",
			existingFinalizers: []string{},
			syncPolicy:         nil,
			expectedFinalizers: []string{"resources-finalizer.argocd.argoproj.io"},
		},
		{
			testName:           "non-nil sync policy should use standard finalizer",
			existingFinalizers: nil,
			syncPolicy:         &argoprojiov1alpha1.ApplicationSetSyncPolicy{},
			expectedFinalizers: []string{"resources-finalizer.argocd.argoproj.io"},
		},
		{
			testName:           "preserveResourcesOnDeletion should not have a finalizer",
			existingFinalizers: nil,
			syncPolicy: &argoprojiov1alpha1.ApplicationSetSyncPolicy{
				PreserveResourcesOnDeletion: true,
			},
			expectedFinalizers: nil,
		},
		{
			testName:           "user-specified finalizer should overwrite preserveResourcesOnDeletion",
			existingFinalizers: []string{"resources-finalizer.argocd.argoproj.io/background"},
			syncPolicy: &argoprojiov1alpha1.ApplicationSetSyncPolicy{
				PreserveResourcesOnDeletion: true,
			},
			expectedFinalizers: []string{"resources-finalizer.argocd.argoproj.io/background"},
		},
	} {

		t.Run(c.testName, func(t *testing.T) {

			// Clone the template application
			application := emptyApplication.DeepCopy()
			application.Finalizers = c.existingFinalizers

			params := map[string]string{
				"one": "two",
			}

			// Render the cloned application, into a new application
			render := Render{}

			res, err := render.RenderTemplateParams(application, nil, c.syncPolicy, params)
			assert.Nil(t, err)

			assert.ElementsMatch(t, res.Finalizers, c.expectedFinalizers)

		})

	}

}
