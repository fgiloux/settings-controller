package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/kcp-dev/logicalcluster/v2"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	cutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	settingsv1alpha1 "github.com/fgiloux/settings-controller/api/v1alpha1"
	apisv1alpha1 "github.com/kcp-dev/kcp/pkg/apis/apis/v1alpha1"
)

type SettingsReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	CtrlConfig      settingsv1alpha1.SettingsConfig
	ExportWorkspace string
	ExportName      string
}

const SettingName = "pipeline-service"
const NpName = "hermetic-build"
const QtName = "settings"
const QuotaAnnotation = "\"experimental.quota.kcp.dev/cluster-scoped\": \"true\""

// +kubebuilder:rbac:groups="networking.k8s.io",resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="networking.k8s.io",resources=networkpolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="networking.k8s.io",resources=networkpolicies/finalizers,verbs=update

// +kubebuilder:rbac:groups="",resources=resourcequotas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=resourcequotas/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=resourcequotas/finalizers,verbs=update

// +kubebuilder:rbac:groups="apis.kcp.dev",resources=apibindings,verbs=get;list;watch
// +kubebuilder:rbac:groups="apis.kcp.dev",resources=apibindings/status,verbs=get
// +kubebuilder:rbac:groups="apis.kcp.dev",resources=apibindings/finalizers,verbs=update

// +kubebuilder:rbac:groups=configuration.pipeline-service.io,resources=settings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=configuration.pipeline-service.io,resources=settings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=configuration.pipeline-service.io,resources=settings/finalizers,verbs=update

func (r *SettingsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	//logger := log.FromContext(ctx)
	logger := ctrl.Log.WithName("settings-reconciler")

	// Include the clusterName from req.ClusterName in the logger, similar to the namespace and name keys that are already
	// there.
	logger = logger.WithValues("clusterName", req.ClusterName)
	logger.V(0).Info("Starting reconcile")

	// Add the logical cluster to the context
	ctx = logicalcluster.WithCluster(ctx, logicalcluster.New(req.ClusterName))

	logger.V(3).Info("Getting APIBinding", "NamespacedName", req.NamespacedName)
	var ab apisv1alpha1.APIBinding
	if err := r.Get(ctx, req.NamespacedName, &ab); err != nil {
		if errors.IsNotFound(err) {
			// Normal - was deleted
			// Rely on owner references for cascading deletion
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Only process the relevant APIBinding
	// It may go away if kcp makes it possible to only watch APIBindings
	// for the specified APIExport.
	if ab.Spec.Reference.Workspace.ExportName != r.ExportName ||
		ab.Spec.Reference.Workspace.Path != r.ExportWorkspace {
		logger.V(3).Info("APIBinding excluded", "NamespacedName", req.NamespacedName)
		return ctrl.Result{}, nil
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

	qtCondition := metav1.Condition{
		Type:   "QuotasReady",
		Status: metav1.ConditionUnknown,
		LastTransitionTime: metav1.Time{
			Time: time.Now().UTC(),
		},
		Reason:  "Unknown",
		Message: "Unknown",
	}

	// get the settings associated with the apibinding
	var s settingsv1alpha1.Settings
	sn := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      SettingName,
	}
	if err := r.Get(ctx, sn, &s); err != nil {
		if errors.IsNotFound(err) {
			// Settings need to be created
			logger.V(3).Info("Settings not found (needs to be created)", "NamespacedName", sn)
			s = settingsv1alpha1.Settings{}
			s.SetName(SettingName)
			// Set the APIBinding instance as the owner and controller
			ctrl.SetControllerReference(&ab, &s, r.Scheme)
			if err = r.Create(ctx, &s); err != nil {
				logger.Error(err, "unable to create settings", "resource", s)
				return ctrl.Result{}, err
			}
			logger.V(1).Info("Settings created")
			return ctrl.Result{Requeue: true}, nil
		} else {
			return ctrl.Result{}, err
		}
	}

	scopy := s.DeepCopy()

	if len(s.Status.Conditions) == 0 {
		patch := client.MergeFrom(scopy)
		s.Status.Conditions = append(s.Status.Conditions, npCondition)
		s.Status.Conditions = append(s.Status.Conditions, qtCondition)
		err := r.Status().Patch(ctx, &s, patch)
		if err != nil {
			logger.Info("Patch error", "error", err)
		}
		return ctrl.Result{Requeue: true}, err
	}

	var ns corev1.Namespace
	if err := r.Get(ctx, types.NamespacedName{Name: r.CtrlConfig.Namespace}, &ns); err != nil {
		if errors.IsNotFound(err) {
			ns.SetName(r.CtrlConfig.Namespace)
			// Set the APIBinding instance as the owner and controller
			ctrl.SetControllerReference(&ab, &ns, r.Scheme)
			if err = r.Create(ctx, &ns); err != nil {
				logger.Error(err, "unable to create namespace", "resource", ns)
				return ctrl.Result{}, err
			}
			logger.V(1).Info("Settings created")
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	npCondition.Reason = "NetworkPoliciesCreated"
	npCondition.Message = fmt.Sprintf("NetworkPolicies successfully created in %q namespace", r.CtrlConfig.Namespace)
	npCondition.Status = metav1.ConditionTrue

	qtCondition.Reason = "QuotasCreated"
	qtCondition.Message = fmt.Sprintf("Quotas successfully created in %q namespace", r.CtrlConfig.Namespace)
	qtCondition.Status = metav1.ConditionTrue

	var rtnErr error

	// A single Quota created in a single namespace defined in the operator configuration
	// The annotation makes the quota cluster scoped.
	// Reverse claim should enforce that the quota cannot be changed by a workspace admin
	// as long the workspace is bound to the apiexport of the controller
	var wsQt corev1.ResourceQuota
	wsQt.SetNamespace(r.CtrlConfig.Namespace)
	wsQt.SetName(QtName)
	wsQt.SetAnnotations(map[string]string{"experimental.quota.kcp.dev/cluster-scoped": "true"})
	// Set the APIBinding instance as the owner and controller
	ctrl.SetControllerReference(&ab, &wsQt, r.Scheme)
	operationResult, rtnErr := cutil.CreateOrPatch(ctx, r.Client, &wsQt, func() error {
		wsQt.Spec = r.CtrlConfig.QuotaConfig.Spec
		return nil
	})
	if rtnErr != nil {
		logger.Error(rtnErr, "unable to create or patch the ResourceQuota")
		qtCondition.Status = metav1.ConditionFalse
		qtCondition.Reason = "Error"
		qtCondition.Message = "Unable to create or patch the ResourceQuota"
	}
	logger.V(2).Info(string(operationResult), "resource", wsQt)

	// A single NetworkPolicy created in a single namespace defined in the operator configuration
	// There is no enforcement, more a feature (hermetic build) than a constraint.
	var wsNP netv1.NetworkPolicy
	wsNP.SetNamespace(r.CtrlConfig.Namespace)
	wsNP.SetName(NpName)
	// Set the APIBinding instance as the owner and controller
	ctrl.SetControllerReference(&ab, &wsNP, r.Scheme)
	operationResult, rtnErr = cutil.CreateOrPatch(ctx, r.Client, &wsNP, func() error {
		wsNP.Spec = r.CtrlConfig.NetPolConfig.Spec
		return nil
	})
	if rtnErr != nil {
		logger.Error(rtnErr, "unable to create or patch the NetworkPolicy")
		npCondition.Status = metav1.ConditionFalse
		npCondition.Reason = "Error"
		npCondition.Message = "Unable to create or patch the NetworkPolicy"
	}
	logger.V(2).Info(string(operationResult), "resource", wsNP)

	// Update the condition only if it is missing, the status or the reason of the available condition has changed.
	npConditionChanged := true
	qtConditionChanged := true
	for i, condition := range s.Status.Conditions {
		if condition.Type == npCondition.Type {
			if condition.Status != npCondition.Status || condition.Reason != npCondition.Reason {
				s.Status.Conditions[i] = npCondition
				continue
			} else {
				npConditionChanged = false
			}
		}
		if condition.Type == qtCondition.Type {
			if condition.Status != qtCondition.Status || condition.Reason != qtCondition.Reason {
				s.Status.Conditions[i] = qtCondition
			} else {
				qtConditionChanged = false
			}
		}
	}

	if npConditionChanged || qtConditionChanged {
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
		For(&apisv1alpha1.APIBinding{}).
		Owns(&settingsv1alpha1.Settings{}).
		Owns(&corev1.ResourceQuota{}).
		Owns(&netv1.NetworkPolicy{}).
		Complete(r)
}
