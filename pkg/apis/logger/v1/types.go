package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Logger struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Status            LoggerStatus `json:"status,omitempty"`
	Spec              LoggerSpec   `json:"spec,omitempty"`
}

type LoggerStatus struct {
	Value StatusValue `json:"state"`
}

type StatusValue string

const (
	Available   StatusValue = "Available"
	Unavailable StatusValue = "Unavailable"
)

type LoggerSpec struct {
	Name         string `json:"name"`
	TimeInterval int    `json:"timeInterval"`
	Replicas     *int32 `json:"replicas"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type LoggerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `son:"metadata,omitempty"`
	Items           []*Logger `json:"items"`
}
