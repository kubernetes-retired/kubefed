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
	extv1b1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FederatedIngressSpec defines the desired state of FederatedIngress
type FederatedIngressSpec struct {
	// Template to derive per-cluster ingress from
	Template extv1b1.Ingress `json:"template,omitempty"`
}

// FederatedIngressStatus defines the observed state of FederatedIngress
type FederatedIngressStatus struct {
	ClusterStatuses []FederatedIngressClusterStatus `json:"clusterStatuses,omitempty"`
}

// FederatedIngressClusterStatus is the observed status for a named cluster
type FederatedIngressClusterStatus struct {
	ClusterName string                `json:"clusterName,omitempty"`
	Status      extv1b1.IngressStatus `json:"status,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FederatedIngress
// +k8s:openapi-gen=true
// +kubebuilder:resource:path=federatedingresses
type FederatedIngress struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FederatedIngressSpec   `json:"spec,omitempty"`
	Status FederatedIngressStatus `json:"status,omitempty"`
}
