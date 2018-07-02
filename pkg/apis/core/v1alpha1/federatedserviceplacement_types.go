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

// FederatedServicePlacementSpec defines the desired state of FederatedServicePlacement
type FederatedServicePlacementSpec struct {
	// Names of the clusters that a federated resource should exist in.
	ClusterNames []string `json:"clusterNames,omitempty"`
}

// FederatedServicePlacementStatus defines the observed state of FederatedServicePlacement
type FederatedServicePlacementStatus struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FederatedServicePlacement
// +k8s:openapi-gen=true
// +kubebuilder:resource:path=federatedserviceplacements
type FederatedServicePlacement struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FederatedServicePlacementSpec   `json:"spec,omitempty"`
	Status FederatedServicePlacementStatus `json:"status,omitempty"`
}
