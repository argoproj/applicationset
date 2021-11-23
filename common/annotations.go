package common

const (
	// AnnotationApplicationRefresh is an annotation that is added when an ApplicationSet is requested to be refreshed by a webhook. The ApplicationSet controller will remove this annotation at the end of reconcilation.
	AnnotationApplicationSetRefresh = "argocd.argoproj.io/application-set-refresh"
)
