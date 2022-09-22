package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cfg "sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"
)

// NOTE: json tags are required. Any new fields you add must have json tags for the fields to be serialized.

type SettingsNetPolConfig struct {
	// Specification of the desired behavior for this NetworkPolicy.
	// +optional
	Spec netv1.NetworkPolicySpec `json:"spec,omitempty"`
}

type SettingsQuotaConfig struct {
	// Defines the desired quota.
	// +optional
	Spec corev1.ResourceQuotaSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// SettingsConfig is the Schema for the settingsconfigs API
type SettingsConfig struct {
	metav1.TypeMeta `json:",inline"`

	// ControllerManagerConfigurationSpec returns the generic configuration for controllers
	cfg.ControllerManagerConfigurationSpec `json:",inline"`

	// Namespace defines the space within which each name must be unique. An empty namespace is
	// equivalent to the "default" namespace, but "default" is the canonical representation.
	// Not all objects are required to be scoped to a namespace - the value of this field for
	// those objects will be empty.
	//
	// Must be a DNS_LABEL.
	// Cannot be updated.
	// More info: http://kubernetes.io/docs/user-guide/namespaces
	// +optional
	Namespace string `json:"namespace,omitempty"`

	NetPolConfig SettingsNetPolConfig `json:"networkPolicyConfig,omitempty"`
	QuotaConfig  SettingsQuotaConfig  `json:"quotaConfig,omitempty"`
}

//+kubebuilder:object:root=true

func init() {
	SchemeBuilder.Register(&SettingsConfig{})
}
