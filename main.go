package main

import (
	"context"
	"flag"
	"github.com/argoproj-labs/applicationset/pkg/generators"
	"github.com/argoproj-labs/applicationset/pkg/services"
	"github.com/argoproj-labs/applicationset/pkg/utils"
	argov1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/cache"

	"os"

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
	var policy string
	var debugLog bool
	var dryRun bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&metricsAddr, "probe-addr", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&namespace, "namespace", "argocd", "Argo CD repo namesapce")
	flag.StringVar(&argocdRepoServer, "argocd-repo-server", "argocd-repo-server:8081", "Argo CD repo server address")
	flag.StringVar(&policy, "policy", "sync", "Modify how application is sync between the generator and the cluster. Default is sync (create & update & delete), options: create-only, create-update (no deletion)")
	flag.BoolVar(&debugLog, "debug", false, "print debug logs")
	flag.BoolVar(&dryRun, "dry-run", false, "Enable dry run mode")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	policyObj, exists := utils.Policies[policy]
	if !exists {
		setupLog.Info("Policy value can be: sync, create-only, create-update")
		os.Exit(1)
	}

	if debugLog {
		log.SetLevel(log.DebugLevel)
	}

	// Determine the namespace we're running in. Normally injected into the pod as an env
	// var via the Kube downward API configured in the Deployment.
	// Developers running the binary locally will need to remember to set the NAMESPACE environment variable.
	ns := os.Getenv("NAMESPACE")
	if len(ns) == 0 {
		setupLog.Info("Please set NAMESPACE environment variable to match where you are running the applicationset controller")
		os.Exit(1)
	}
	setupLog.Info("using argocd namespace", "namespace", ns)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		// Our cache and thus watches and client queries are restricted to the namespace we're running in. This assumes
		// the applicationset controller is in the same namespace as argocd, which should be the same namespace of
		// all cluster Secrets and Applications we interact with.
		NewCache:               cache.MultiNamespacedCacheBuilder([]string{ns}),
		HealthProbeBindAddress: probeBindAddr,
		Port:                   9443,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "58ac56fa.",
		DryRunClient:           dryRun,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	k8s := kubernetes.NewForConfigOrDie(mgr.GetConfig())

	if err = (&controllers.ApplicationSetReconciler{
		Generators: map[string]generators.Generator{
			"List":     generators.NewListGenerator(),
			"Clusters": generators.NewClusterGenerator(mgr.GetClient()),
			"Git":      generators.NewGitGenerator(services.NewArgoCDService(context.Background(), k8s, namespace, argocdRepoServer)),
		},
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("applicationset-controller"),
		Renderer: &utils.Render{},
		Policy:   policyObj,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ApplicationSet")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
