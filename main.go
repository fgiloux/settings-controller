package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/kcp"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	settingsv1alpha1 "github.com/fgiloux/settings-controller/api/v1alpha1"
	// +kubebuilder:scaffold:imports

	"github.com/fgiloux/settings-controller/controllers"

	apisv1alpha1 "github.com/kcp-dev/kcp/pkg/apis/apis/v1alpha1"
)

var (
	scheme            = runtime.NewScheme()
	setupLog          = ctrl.Log.WithName("setup")
	kubeconfigContext string
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apisv1alpha1.AddToScheme(scheme))
	utilruntime.Must(settingsv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme

	flag.StringVar(&kubeconfigContext, "context", "", "kubeconfig context")
}

func main() {
	var configFile string
	var metricsAddr string
	var enableLeaderElection bool
	var leaderElectionNs string
	var probeAddr string
	var apiExportName string
	var apiExportWs string
	// The file configuration takes precedence over the flags and their default values.
	flag.StringVar(&configFile, "config", "config/manager/controller_manager_config.yaml", "The controller will load its initial configuration from this file. "+
		"Omit this flag to use the default configuration values. "+
		"Command-line flags override configuration from this file.")
	flag.StringVar(&apiExportName, "api-export-name", "settings-configuration.pipeline-service.io", "The name of the APIExport.")
	flag.StringVar(&apiExportWs, "api-export-workspace", "", "The workspace containing the APIExport.")
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&leaderElectionNs, "leader-elect-ns", "", "The namespace used for leader election")
	logOpts := zap.Options{
		Development: true,
	}

	logOpts.BindFlags(flag.CommandLine)
	klog.InitFlags(flag.CommandLine)

	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&logOpts)))

	ctx := ctrl.SetupSignalHandler()

	restConfig := ctrl.GetConfigOrDie()

	setupLog = setupLog.WithValues("api-export-name", apiExportName)

	/*if err := apisv1alpha1.AddToScheme(scheme); err != nil {
		setupLog.Error(err, "error adding apis.kcp.dev/v1alpha1 to scheme")
		os.Exit(1)
	}*/

	var mgr ctrl.Manager
	var err error
	ctrlConfig := settingsv1alpha1.SettingsConfig{}
	options := ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		//		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		// LeaderElectionID:       "67a0541b.pipeline-service.io",
		LeaderElectionNamespace: leaderElectionNs,
		LeaderElectionConfig:    restConfig,
	}

	if configFile != "" {
		options, err = options.AndFrom(ctrl.ConfigFile().AtPath(configFile).OfKind(&ctrlConfig))
		if err != nil {
			setupLog.Error(err, "unable to load the config file")
			os.Exit(1)
		}
	}

	setupLog.V(1).Info("Looking up virtual workspace URL")
	cfg, err := restConfigForAPIExport(ctx, restConfig, apiExportName)
	if err != nil {
		setupLog.Error(err, "error looking up virtual workspace URL")
		os.Exit(1)
	}

	setupLog.Info("Using virtual workspace URL", "url", cfg.Host)

	options.LeaderElectionConfig = restConfig
	mgr, err = kcp.NewClusterAwareManager(cfg, options)
	if err != nil {
		setupLog.Error(err, "unable to start cluster aware manager")
		os.Exit(1)
	}

	if err = (&controllers.SettingsReconciler{
		Client:          mgr.GetClient(),
		Scheme:          mgr.GetScheme(),
		CtrlConfig:      ctrlConfig,
		ExportWorkspace: apiExportWs,
		ExportName:      apiExportName,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Settings")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")

	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// +kubebuilder:rbac:groups="apis.kcp.dev",resources=apiexports,verbs=get;list;watch

// restConfigForAPIExport returns a *rest.Config properly configured to communicate with the endpoint for the
// APIExport's virtual workspace.
func restConfigForAPIExport(ctx context.Context, cfg *rest.Config, apiExportName string) (*rest.Config, error) {

	scheme := runtime.NewScheme()
	if err := apisv1alpha1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("error adding apis.kcp.dev/v1alpha1 to scheme: %w", err)
	}

	apiExportClient, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("error creating APIExport client: %w", err)
	}

	var apiExport apisv1alpha1.APIExport

	if apiExportName != "" {
		if err := apiExportClient.Get(ctx, types.NamespacedName{Name: apiExportName}, &apiExport); err != nil {
			return nil, fmt.Errorf("error getting APIExport %q: %w", apiExportName, err)
		}
	} else {
		setupLog.Info("api-export-name is empty - listing")
		exports := &apisv1alpha1.APIExportList{}
		if err := apiExportClient.List(ctx, exports); err != nil {
			return nil, fmt.Errorf("error listing APIExports: %w", err)
		}
		if len(exports.Items) == 0 {
			return nil, fmt.Errorf("no APIExport found")
		}
		if len(exports.Items) > 1 {
			return nil, fmt.Errorf("more than one APIExport found")
		}
		apiExport = exports.Items[0]
	}

	if len(apiExport.Status.VirtualWorkspaces) < 1 {
		return nil, fmt.Errorf("APIExport %q status.virtualWorkspaces is empty", apiExportName)
	}

	cfg = rest.CopyConfig(cfg)
	// TODO(ncdc): sharding support
	cfg.Host = apiExport.Status.VirtualWorkspaces[0].URL

	return cfg, nil
}
