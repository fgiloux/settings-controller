package controllers

import (
	"context"
	"strconv"
	"time"

	"github.com/kcp-dev/logicalcluster"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	settingsv1alpha1 "github.com/fgiloux/settings-controller/api/v1alpha1"
)

// SettingsReconciler reconciles a Settings object
type SettingsReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=configuration.pipeline-service.io,resources=settings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=configuration.pipeline-service.io,resources=settings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=configuration.pipeline-service.io,resources=settings/finalizers,verbs=update

// Reconcile TODO
func (r *SettingsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Include the clusterName from req.ObjectKey in the logger, similar to the namespace and name keys that are already
	// there.
	logger = logger.WithValues("clusterName", req.ClusterName)

	// You probably wouldn't need to do this, but if you wanted to list all instances across all logical clusters:
	var allSettings settingsv1alpha1.SettingsList
	if err := r.List(ctx, &allSettings); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Listed all Settings across all workspaces", "count", len(allSettings.Items))

	// Add the logical cluster to the context
	ctx = logicalcluster.WithCluster(ctx, logicalcluster.New(req.ClusterName))

	logger.Info("Getting Settings")
	var s settingsv1alpha1.Settings
	if err := r.Get(ctx, req.NamespacedName, &s); err != nil {
		if errors.IsNotFound(err) {
			// Normal - was deleted
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	logger.Info("Listing all Settings in the current logical cluster")
	var list settingsv1alpha1.SettingsList
	if err := r.List(ctx, &list); err != nil {
		return ctrl.Result{}, err
	}

	numSettings := len(list.Items)
	totalCondition := metav1.Condition{
		Type:   "Total",
		Status: metav1.ConditionTrue,
		LastTransitionTime: metav1.Time{
			Time: time.Now().UTC(),
		},
		Reason:  "total:" + strconv.Itoa(numSettings),
		Message: "Counting",
	}
	scopy := s.DeepCopy()
	totalNew := true
	changed := false
	// Update the condition only if the status (or count....it is a hack) of the available condition has changed.
	for i, condition := range s.Status.Conditions {
		if condition.Type == totalCondition.Type {
			totalNew = false
			if condition.Status != totalCondition.Status || condition.Reason != totalCondition.Reason {
				s.Status.Conditions[i] = totalCondition
				changed = true
				break
			}
		}
	}
	if totalNew {
		s.Status.Conditions = append(s.Status.Conditions, totalCondition)
		changed = true
	}

	if changed {
		logger.Info("Patching Settings status to store total Settings count in the current logical cluster")
		patch := client.MergeFrom(scopy)

		if err := r.Status().Patch(ctx, &s, patch); err != nil {
			logger.Info("Patch error", "error", err)
			// TODO: depending on the error I may just give up
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SettingsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&settingsv1alpha1.Settings{}).
		Complete(r)
}
