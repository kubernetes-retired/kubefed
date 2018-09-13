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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FederatedServiceAccountSpec defines the desired state of FederatedServiceAccount
type FederatedServiceAccountSpec struct {
	// Template to derive per-cluster service account from
	Template corev1.ServiceAccount `json:"template,omitempty"`
}

// FederatedServiceAccountStatus defines the observed state of FederatedServiceAccount
type FederatedServiceAccountStatus struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FederatedServiceAccount
// +k8s:openapi-gen=true
// +kubebuilder:resource:path=federatedserviceaccounts
type FederatedServiceAccount struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FederatedServiceAccountSpec   `json:"spec,omitempty"`
	Status FederatedServiceAccountStatus `json:"status,omitempty"`
}
