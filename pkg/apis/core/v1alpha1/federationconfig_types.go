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

// EDIT THIS FILE!
// Created by "kubebuilder create resource" for you to implement the FederationConfig resource schema definition
// as a go struct.
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FederationConfigSpec defines the desired state of FederationConfig
type FederationConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "kubebuilder generate" to regenerate code after modifying this file

	LimitedScope       bool                 `json:"limited-scope,omitempty"`
	RegistryNamespace  string               `json:"registry-namespace,omitempty"`
	ControllerDuration DurationConfig       `json:"controller-duration,omitempty"`
	LeaderElect        LeaderElectConfig    `json:"leader-elect,omitempty"`
	FeatureGates       []FeatureGatesConfig `json:"feature-gates,omitempty"`
}

type DurationConfig struct {
	AvailableDelay       metav1.Duration `json:"available-delay,omitempty"`
	UnavailableDelay     metav1.Duration `json:"unavailable-delay,omitempty"`
	ClusterMonitorPeriod metav1.Duration `json:"cluster-monitor-period,omitempty"`
}
type LeaderElectConfig struct {
	LeaseDuration metav1.Duration `json:"lease-duration,omitempty"`
	RenewDeadline metav1.Duration `json:"renew-deadline,omitempty"`
	RetryPeriod   metav1.Duration `json:"retry-period,omitempty"`
	ResourceLock  string          `json:"resource-lock,omitempty"`
}
type FeatureGatesConfig struct {
	Name    string `json:"name,omitempty"`
	Enabled bool   `json:"enabled,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FederationConfig
// +k8s:openapi-gen=true
// +kubebuilder:resource:path=federationconfigs
type FederationConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec FederationConfigSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FederationConfigList contains a list of FederationConfig
type FederationConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FederationConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FederationConfig{}, &FederationConfigList{})
}
