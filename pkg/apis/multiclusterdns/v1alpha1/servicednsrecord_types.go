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

// ServiceDNSRecordSpec defines the desired state of ServiceDNSRecord
type ServiceDNSRecordSpec struct {
	// FederationName is the name of the federation to which the corresponding federated service belongs
	FederationName string `json:"federationName,omitempty"`
	// DNSSuffix is the suffix (domain) to append to DNS names
	DNSSuffix string `json:"dnsSuffix,omitempty"`
	// RecordTTL is the TTL in seconds for DNS records created for this Service, if omitted a default would be used
	RecordTTL TTL `json:"recordTTL,omitempty"`
}

// ServiceDNSRecordStatus defines the observed state of ServiceDNSRecord
type ServiceDNSRecordStatus struct {
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

// ServiceDNSRecord
// +k8s:openapi-gen=true
// +kubebuilder:resource:path=servicednsrecords
// +kubebuilder:subresource:status
type ServiceDNSRecord struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceDNSRecordSpec   `json:"spec,omitempty"`
	Status ServiceDNSRecordStatus `json:"status,omitempty"`
}
