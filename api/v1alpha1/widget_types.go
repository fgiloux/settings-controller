package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WidgetSpec defines the desired state of Widget
type WidgetSpec struct {
	Foo string `json:"foo,omitempty"`
}

// WidgetStatus defines the observed state of Widget
type WidgetStatus struct {
	Total int `json:"total,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Widget is the Schema for the widgets API
type Widget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WidgetSpec   `json:"spec,omitempty"`
	Status WidgetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WidgetList contains a list of Widget
type WidgetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Widget `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Widget{}, &WidgetList{})
}
