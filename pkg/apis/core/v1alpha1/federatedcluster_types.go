/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/common"
)

// FederatedClusterSpec defines the desired state of FederatedCluster
type FederatedClusterSpec struct {
	// Name of the cluster registry Cluster resource from which to source api
	// endpoints.
	// TODO(marun) should this go away in favor of a 1:1 mapping?
	ClusterRef apiv1.LocalObjectReference `json:"clusterRef,omitempty"`

	// Name of the secret containing kubeconfig to access the referenced cluster.
	//
	// Admin needs to ensure that the required secret exists. Secret
	// should be in the same namespace where federation control plane
	// is hosted and it should have kubeconfig in its data with key
	// "kubeconfig".
	//
	// This will later be changed to a reference to secret in
	// federation control plane when the federation control plane
	// supports secrets.
	//
	// This can be left empty if the cluster allows insecure access.
	// +optional
	SecretRef *apiv1.LocalObjectReference `json:"secretRef,omitempty"`
}

// FederatedClusterStatus contains information about the current status of a
// cluster updated periodically by cluster controller.
type FederatedClusterStatus struct {
	// Conditions is an array of current cluster conditions.
	// +optional
	Conditions []ClusterCondition `json:"conditions,omitempty"`
	// Zone is the name of availability zone in which the nodes of the cluster exist, e.g. 'us-east1-a'.
	// +optional
	Zone string `json:"zone,omitempty"`
	// Region is the name of the region in which all of the nodes in the cluster exist.  e.g. 'us-east1'.
	// +optional
	Region string `json:"region,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FederatedCluster configures federation to be aware of a Kubernetes cluster
// from the cluster-registry and provides a Kubeconfig for federation to use to
// communicate with the cluster.
//
// +k8s:openapi-gen=true
// +kubebuilder:resource:path=federatedclusters
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name=ready,type=string,JSONPath=.status.conditions[?(@.type=='Ready')].status
type FederatedCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FederatedClusterSpec   `json:"spec,omitempty"`
	Status FederatedClusterStatus `json:"status,omitempty"`
}

// ClusterCondition describes current state of a cluster.
type ClusterCondition struct {
	// Type of cluster condition, Ready or Offline.
	Type common.ClusterConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status apiv1.ConditionStatus `json:"status"`
	// Last time the condition was checked.
	// +optional
	LastProbeTime metav1.Time `json:"lastProbeTime,omitempty"`
	// Last time the condition transit from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// (brief) reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// Human readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FederatedClusterList contains a list of FederatedCluster
type FederatedClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FederatedCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FederatedCluster{}, &FederatedClusterList{})
}
