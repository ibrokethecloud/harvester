package v1beta1

import (
	lhv1beta2 "github.com/longhorn/longhorn-manager/k8s/pkg/apis/longhorn/v1beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.backupStatus`
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status

type SystemBackup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            SystemBackupStatus `json:"status,omitempty"`
}

type SystemBackupStatus struct {
	LonghornSystemBackupStatus *lhv1beta2.SystemBackupStatus `json:"lhSystemBackupStatus,omitempty"`
}
