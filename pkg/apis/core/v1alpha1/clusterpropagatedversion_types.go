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

// ClusterPropagatedVersionSpec defines the desired state of ClusterPropagatedVersion
type ClusterPropagatedVersionSpec struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced

// ClusterPropagatedVersion holds version information about the state
// propagated from cluster-scoped federation APIs configured by
// FederatedTypeConfig to target clusters. The name of a
// ClusterPropagatedVersion encodes the kind and name of the resource
// it stores information for. The type of version information stored
// in ClusterPropagatedVersion will be the metadata.resourceVersion or
// metadata.Generation of the resource depending on the value of
// spec.comparisonField in the FederatedTypeConfig associated with the
// resource.
//
// +k8s:openapi-gen=true
// +kubebuilder:resource:path=clusterpropagatedversions
// +kubebuilder:subresource:status
type ClusterPropagatedVersion struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterPropagatedVersionSpec `json:"spec,omitempty"`
	Status PropagatedVersionStatus      `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced

// ClusterPropagatedVersionList contains a list of ClusterPropagatedVersion
type ClusterPropagatedVersionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterPropagatedVersion `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterPropagatedVersion{}, &ClusterPropagatedVersionList{})
}
