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
	apiextv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FederationConfigSpec defines the desired state of FederationConfig
type FederationConfigSpec struct {
	// The scope of the federation control plane should be either `Namespaced` or `Cluster`.
	// `Namespaced` indicates that the federation namespace will be the only target for federation.
	Scope apiextv1b1.ResourceScope `json:"scope,omitempty"`
	// The cluster registry namespace.
	RegistryNamespace  string                   `json:"registry-namespace,omitempty"`
	ControllerDuration DurationConfig           `json:"controller-duration,omitempty"`
	LeaderElect        LeaderElectConfig        `json:"leader-elect,omitempty"`
	FeatureGates       []FeatureGatesConfig     `json:"feature-gates,omitempty"`
	ClusterHealthCheck ClusterHealthCheckConfig `json:"cluster-health-check,omitempty"`
}

type DurationConfig struct {
	// Time to wait before reconciling on a healthy cluster.
	AvailableDelay metav1.Duration `json:"available-delay,omitempty"`
	// Time to wait before giving up on an unhealthy cluster.
	UnavailableDelay metav1.Duration `json:"unavailable-delay,omitempty"`
}
type LeaderElectConfig struct {
	// The duration that non-leader candidates will wait after observing a leadership
	// renewal until attempting to acquire leadership of a led but unrenewed leader
	// slot. This is effectively the maximum duration that a leader can be stopped
	// before it is replaced by another candidate. This is only applicable if leader
	// election is enabled.
	LeaseDuration metav1.Duration `json:"lease-duration,omitempty"`
	// The interval between attempts by the acting master to renew a leadership slot
	// before it stops leading. This must be less than or equal to the lease duration.
	// This is only applicable if leader election is enabled.
	RenewDeadline metav1.Duration `json:"renew-deadline,omitempty"`
	// The duration the clients should wait between attempting acquisition and renewal
	// of a leadership. This is only applicable if leader election is enabled.
	RetryPeriod metav1.Duration `json:"retry-period,omitempty"`
	// The type of resource object that is used for locking during
	// leader election. Supported options are `configmaps` (default) and `endpoints`.
	ResourceLock string `json:"resource-lock,omitempty"`
}
type FeatureGatesConfig struct {
	Name    string `json:"name,omitempty"`
	Enabled bool   `json:"enabled,omitempty"`
}

type ClusterHealthCheckConfig struct {
	// How often to monitor the cluster health (in seconds).
	PeriodSeconds int `json:"period-seconds,omitempty"`
	// Minimum consecutive failures for the cluster health to be considered failed after having succeeded.
	FailureThreshold int `json:"failure-threshold,omitempty"`
	// Minimum consecutive successes for the cluster health to be considered successful after having failed.
	SuccessThreshold int `json:"success-threshold,omitempty"`
	// Number of seconds after which the cluster health check times out.
	TimeoutSeconds int `json:"timeout-seconds,omitempty"`
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
