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

// JobSchedulingPreferenceSpec defines the desired state of JobSchedulingPreference
type JobSchedulingPreferenceSpec struct {
	// Specifies the maximum desired number of pods this FederatedJob should
	// run at any given time.
	// Parallelism specified in the spec for target job template will be
	// discarded/overridden when job scheduling preferences are specified.
	// // TODO(irfanurrehman): If needed fix up a strategy about creating jobs
	// into federated clusters with parallelism as 0. Right now we do not
	// create jobs in those clusters which have 0 weightage in ClusterWeights.
	TotalParallelism int32 `json:"totalParallelism"`

	// Specifies the desired number of successfully finished pods this
	// FederatedJob should be run with.
	// Completions specified in the spec for target job template will be
	// discarded/overridden when job scheduling preferences are specified.
	TotalCompletions int32 `json:"totalCompletions"`

	// A weight ratio specification per cluster. The same weight value will be applicable to
	// both parallelism and completions per job per cluster. The distribution of parallelism
	// and completions will be done using weightN/sumWeights.
	// If omitted for a particular cluster(s), cluster(s) without explicit weight will not
	// have any jobs scheduled.
	// +optional
	ClusterWeights map[string]int32 `json:"clusterWeights,omitempty"`
}

// JobSchedulingPreferenceStatus defines the observed state of JobSchedulingPreference
type JobSchedulingPreferenceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "kubebuilder generate" to regenerate code after modifying this file
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JobSchedulingPreference
// +k8s:openapi-gen=true
// +kubebuilder:resource:path=jobschedulingpreferences
type JobSchedulingPreference struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JobSchedulingPreferenceSpec   `json:"spec,omitempty"`
	Status JobSchedulingPreferenceStatus `json:"status,omitempty"`
}
