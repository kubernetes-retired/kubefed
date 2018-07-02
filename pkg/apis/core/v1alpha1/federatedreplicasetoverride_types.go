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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FederatedReplicaSetOverrideSpec defines the desired state of FederatedReplicaSetOverride
type FederatedReplicaSetOverrideSpec struct {
	Overrides []FederatedReplicaSetClusterOverride `json:"overrides,omitempty"`
}

// FederatedReplicaSetClusterOverride defines the overrides for a named cluster
type FederatedReplicaSetClusterOverride struct {
	// TODO(marun) Need to ensure that a cluster name only appears
	// once.  Why can't maps be used so this validation is automatic?
	ClusterName string `json:"clusterName,omitempty"`
	Replicas    *int32 `json:"replicas,omitempty"`
}

// FederatedReplicaSetOverrideStatus defines the observed state of FederatedReplicaSetOverride
type FederatedReplicaSetOverrideStatus struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FederatedReplicaSetOverride
// +k8s:openapi-gen=true
// +kubebuilder:resource:path=federatedreplicasetoverrides
type FederatedReplicaSetOverride struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FederatedReplicaSetOverrideSpec   `json:"spec,omitempty"`
	Status FederatedReplicaSetOverrideStatus `json:"status,omitempty"`
}
