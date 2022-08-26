package controllers

import (
	"context"
	"time"

	"github.com/kcp-dev/logicalcluster/v2"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	cutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	settingsv1alpha1 "github.com/fgiloux/settings-controller/api/v1alpha1"
)

// SettingsReconciler reconciles a Settings object
type SettingsReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	CtrlConfig settingsv1alpha1.SettingsConfig
}

// +kubebuilder:rbac:groups="networking.k8s.io",resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="networking.k8s.io",resources=networkpolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="networking.k8s.io",resources=networkpolicies/finalizers,verbs=update

// +kubebuilder:rbac:groups=configuration.pipeline-service.io,resources=settings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=configuration.pipeline-service.io,resources=settings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=configuration.pipeline-service.io,resources=settings/finalizers,verbs=update

func (r *SettingsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Include the clusterName from req.ObjectKey in the logger, similar to the namespace and name keys that are already
	// there.
	logger = logger.WithValues("clusterName", req.ClusterName)

	// Add the logical cluster to the context
	ctx = logicalcluster.WithCluster(ctx, logicalcluster.New(req.ClusterName))

	logger.V(3).Info("Getting Settings")
	var s settingsv1alpha1.Settings
	if err := r.Get(ctx, req.NamespacedName, &s); err != nil {
		if errors.IsNotFound(err) {
			// Normal - was deleted
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	npCondition := metav1.Condition{
		Type:   "NetworkPoliciesReady",
		Status: metav1.ConditionUnknown,
		LastTransitionTime: metav1.Time{
			Time: time.Now().UTC(),
		},
		Reason:  "Unknown",
		Message: "Unknown",
	}

	scopy := s.DeepCopy()

	if len(s.Status.Conditions) == 0 {
		patch := client.MergeFrom(scopy)
		s.Status.Conditions = append(s.Status.Conditions, npCondition)
		err := r.Status().Patch(ctx, &s, patch)
		if err != nil {
			logger.Info("Patch error", "error", err)
		}
		return ctrl.Result{Requeue: true}, err
	}

	npCondition.Reason = "NetworkPoliciesCreated"
	npCondition.Message = "NetworkPolicies for pipeline-service have been successfully created in kcp-system namespace"
	npCondition.Status = metav1.ConditionTrue

	conditionNew := true
	conditionChanged := false
	var rtnErr error

	// TODO: a single NP for a single namespace for now.
	// The current limiting design is not to allow users to CRUD namespaces
	// TODO: namespace should be named pipeline-service, would need to be created
	var wsNP netv1.NetworkPolicy
	wsNP.SetNamespace("kcp-system")
	wsNP.SetName("platform")
	wsNP.SetOwnerReferences([]metav1.OwnerReference{metav1.OwnerReference{
		Name:       s.GetName(),
		UID:        s.GetUID(),
		APIVersion: "v1alpha1",
		Kind:       "Settings",
		Controller: func() *bool { x := true; return &x }(),
	}})
	operationResult, rtnErr := cutil.CreateOrPatch(ctx, r.Client, &wsNP, func() error {
		wsNP.Spec = netv1.NetworkPolicySpec{
			PolicyTypes: []netv1.PolicyType{"Egress"},
			Egress:      r.CtrlConfig.NetPolConfig.Egress,
		}
		return nil
	})
	if rtnErr != nil {
		logger.Error(rtnErr, "unable to create or patch the NetworkPolicy")
		npCondition.Status = metav1.ConditionFalse
		npCondition.Reason = "Error"
		npCondition.Message = "Unable to create or patch the NetworkPolicy"
	}
	logger.V(2).Info(string(operationResult), "networkPolicy", wsNP.GetName())

	// Update the condition only if it is missing or the status of the available condition has changed.
	for i, condition := range s.Status.Conditions {
		if condition.Type == npCondition.Type {
			conditionNew = false
			if condition.Status != npCondition.Status || condition.Reason != npCondition.Reason {
				s.Status.Conditions[i] = npCondition
				conditionChanged = true
				break
			}
		}
	}
	if conditionNew {
		s.Status.Conditions = append(s.Status.Conditions, npCondition)
		conditionChanged = true
	}

	if conditionChanged {
		logger.V(3).Info("Patching Settings status to store the new condition(s) in the current logical cluster")
		patch := client.MergeFrom(scopy)

		if err := r.Status().Patch(ctx, &s, patch); err != nil {
			logger.Info("Patch error", "error", err)
			// TODO: depending on the error it may be better to just give up
			if rtnErr == nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, rtnErr
}

// SetupWithManager sets up the controller with the Manager.
func (r *SettingsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&settingsv1alpha1.Settings{}).
		Owns(&netv1.NetworkPolicy{}).
		Complete(r)
}
