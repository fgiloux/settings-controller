package controllers

import (
	"context"

	"github.com/kcp-dev/logicalcluster"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	settingsv1alpha1 "github.com/openshift-pipelines/pipeline-service/settings-controller/api/v1alpha1"
)

// WidgetReconciler reconciles a Widget object
type WidgetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=configuration.pipeline-service.io,resources=widgets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=configuration.pipeline-service.io,resources=widgets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=configuration.pipeline-service.io,resources=widgets/finalizers,verbs=update

// Reconcile TODO
func (r *WidgetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Include the clusterName from req.ObjectKey in the logger, similar to the namespace and name keys that are already
	// there.
	logger = logger.WithValues("clusterName", req.ClusterName)

	// You probably wouldn't need to do this, but if you wanted to list all instances across all logical clusters:
	var allWidgets settingsv1alpha1.WidgetList
	if err := r.List(ctx, &allWidgets); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Listed all widgets across all workspaces", "count", len(allWidgets.Items))

	// Add the logical cluster to the context
	ctx = logicalcluster.WithCluster(ctx, logicalcluster.New(req.ClusterName))

	logger.Info("Getting widget")
	var w settingsv1alpha1.Widget
	if err := r.Get(ctx, req.NamespacedName, &w); err != nil {
		if errors.IsNotFound(err) {
			// Normal - was deleted
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	logger.Info("Listing all widgets in the current logical cluster")
	var list settingsv1alpha1.WidgetList
	if err := r.List(ctx, &list); err != nil {
		return ctrl.Result{}, err
	}

	numWidgets := len(list.Items)

	if numWidgets == w.Status.Total {
		logger.Info("No need to patch because the widget status is already correct")
		return ctrl.Result{}, nil
	}

	logger.Info("Patching widget status to store total widget count in the current logical cluster")
	original := w.DeepCopy()
	patch := client.MergeFrom(original)

	w.Status.Total = numWidgets

	if err := r.Status().Patch(ctx, &w, patch); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WidgetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&settingsv1alpha1.Widget{}).
		Complete(r)
}
