package main

import (
	"context"
	"flag"
	"github.com/argoproj-labs/applicationset/pkg/generators"
	"github.com/argoproj-labs/applicationset/pkg/services"
	argov1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/google/martian/log"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/argoproj-labs/applicationset/pkg/controllers"
	"k8s.io/client-go/kubernetes"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = argoprojiov1alpha1.AddToScheme(scheme)

	_ = argov1alpha1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var probeBindAddr string
	var enableLeaderElection bool
	var namespace string
	var argocdRepoServer string
	var duration int64
	var debugLog bool
	var updateSync bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&metricsAddr, "probe-addr", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&namespace, "namespace", "argocd", "Argo CD repo namesapce")
	flag.StringVar(&argocdRepoServer, "argocd-repo-server", "argocd-repo-server:8081", "Argo CD repo server address")
	flag.Int64Var(&duration, "git-refresh-duration", 60, "(seconds) The refrash duration for the git generator")
	flag.BoolVar(&debugLog, "debug", false, "print debug logs")
	flag.BoolVar(&updateSync, "sync-update", true, "if false, then the controller will only create and delete application, and will not update existing applications.")


	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	if debugLog {
		log.SetLevel( log.Debug)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		HealthProbeBindAddress: probeBindAddr,
		Port:                   9443,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "58ac56fa.",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	k8s := kubernetes.NewForConfigOrDie(mgr.GetConfig())

	events := make(chan event.GenericEvent)

	stop := ctrl.SetupSignalHandler()

	if err = (&controllers.ApplicationSetReconciler{
		Generators: []generators.Generator{
			generators.NewListGenerator(),
			generators.NewClusterGenerator(mgr.GetClient()),
			generators.NewGitGenerator(services.NewArgoCDService(context.Background(), k8s, namespace, argocdRepoServer)),
		},
		Client:      mgr.GetClient(),
		Scheme:      mgr.GetScheme(),
		Recorder:    mgr.GetEventRecorderFor("applicationset-controller"),
		AppsService: services.NewArgoCDService(context.Background(), k8s, namespace, argocdRepoServer),
		GitRefreshDuration: time.Duration(duration) *time.Second,
		UpdateSync: updateSync,
	}).SetupWithManager(mgr, events); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ApplicationSet")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(stop); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}

}
