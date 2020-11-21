package utils

import (
	"encoding/json"
	"fmt"
	argov1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/pkg/errors"
	"github.com/valyala/fasttemplate"
	"io"
	"strconv"
)

type Renderer interface {
	RenderTemplateParams(tmpl *argov1alpha1.Application, params map[string]string) (*argov1alpha1.Application, error)
}

type Render struct {
}

func (r *Render) RenderTemplateParams(tmpl *argov1alpha1.Application, params map[string]string) (*argov1alpha1.Application, error) {
	if tmpl == nil {
		return nil, fmt.Errorf("Application template is empty ")
	}

	if len(params) == 0 {
		return tmpl, nil
	}

	tmplBytes, err := json.Marshal(tmpl)
	if err != nil {
		return nil, err
	}

	fstTmpl := fasttemplate.New(string(tmplBytes), "{{", "}}")
	replacedTmplStr, err := r.replace(fstTmpl, params, true)
	if err != nil {
		return nil, err
	}

	var replacedTmpl argov1alpha1.Application
	err = json.Unmarshal([]byte(replacedTmplStr), &replacedTmpl)
	if err != nil {
		return nil, err
	}

	if replacedTmpl.ObjectMeta.Finalizers == nil {
		replacedTmpl.ObjectMeta.Finalizers = []string{}
	}
	replacedTmpl.ObjectMeta.Finalizers = append(replacedTmpl.ObjectMeta.Finalizers, "resources-finalizer.argocd.argoproj.io")

	return &replacedTmpl, nil
}

// Replace executes basic string substitution of a template with replacement values.
// allowUnresolved indicates whether or not it is acceptable to have unresolved variables
// remaining in the substituted template. prefixFilter will apply the replacements only
// to variables with the specified prefix
func (r *Render) replace(fstTmpl *fasttemplate.Template, replaceMap map[string]string, allowUnresolved bool) (string, error) {
	var unresolvedErr error
	replacedTmpl := fstTmpl.ExecuteFuncString(func(w io.Writer, tag string) (int, error) {
		replacement, ok := replaceMap[tag]
		if !ok {
			if allowUnresolved {
				// just write the same string back
				return w.Write([]byte(fmt.Sprintf("{{%s}}", tag)))
			}
			unresolvedErr = errors.Errorf("failed to resolve {{%s}}", tag)
			return 0, nil
		}
		// The following escapes any special characters (e.g. newlines, tabs, etc...)
		// in preparation for substitution
		replacement = strconv.Quote(replacement)
		replacement = replacement[1 : len(replacement)-1]
		return w.Write([]byte(replacement))
	})
	if unresolvedErr != nil {
		return "", unresolvedErr
	}

	return replacedTmpl, nil
}
