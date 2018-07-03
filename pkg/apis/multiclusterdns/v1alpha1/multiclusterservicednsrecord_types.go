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

// MultiClusterServiceDNSRecordSpec defines the desired state of MultiClusterServiceDNSRecord
type MultiClusterServiceDNSRecordSpec struct {
	// FederationName is the name of the federation to which the corresponding federated service belongs
	FederationName string `json:"federationName,omitempty"`
	// DNSSuffix is the suffix (domain) to append to DNS names
	DNSSuffix string `json:"dnsSuffix,omitempty"`
}

// MultiClusterServiceDNSRecordStatus defines the observed state of MultiClusterServiceDNSRecord
type MultiClusterServiceDNSRecordStatus struct {
	DNS []ClusterDNS `json:"dns,omitempty"`
}

// ClusterDNS defines the observed status of LoadBalancer within a cluster.
type ClusterDNS struct {
	// Cluster name
	Cluster string `json:"cluster,omitempty"`
	// LoadBalancer for the corresponding service
	LoadBalancer corev1.LoadBalancerStatus `json:"loadBalancer,omitempty"`
	// Zone to which the cluster belongs
	Zone string `json:"zone,omitempty"`
	// Region to which the cluster belongs
	Region string `json:"region,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MultiClusterServiceDNSRecord
// +k8s:openapi-gen=true
// +kubebuilder:resource:path=multiclusterservicednsrecords
// +kubebuilder:subresource:status
type MultiClusterServiceDNSRecord struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MultiClusterServiceDNSRecordSpec   `json:"spec,omitempty"`
	Status MultiClusterServiceDNSRecordStatus `json:"status,omitempty"`
}
