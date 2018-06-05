/*
Copyright 2018 The Federation v2 Authors.

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
	"log"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/federatedscheduling"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ReplicaSchedulingPreference
// +k8s:openapi-gen=true
// +resource:path=replicaschedulingpreferences,strategy=ReplicaSchedulingPreferenceStrategy
type ReplicaSchedulingPreference struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ReplicaSchedulingPreferenceSpec   `json:"spec,omitempty"`
	Status ReplicaSchedulingPreferenceStatus `json:"status,omitempty"`
}

// ObjectReference contains enough information to let you identify the referred resource.
// Currently supported kinds for this type in federation will be FederatedDeployments
// and FederatedReplicasets
type ObjectReference struct {
	// Kind of the referent; More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds"
	Kind string
	// Name of the referent; More info: http://kubernetes.io/docs/user-guide/identifiers#names
	Name string
}

// ReplicaSchedulingPreferenceSpec defines the desired state of ReplicaSchedulingPreference
type ReplicaSchedulingPreferenceSpec struct {
	//TODO (@irfanurrehman); upgrade this to label selector if need be.
	PreferenceTargetRef ObjectReference

	// Total number of pods desired across federated clusters.
	// Replicas specified in the spec for target deployment template or replicaset
	// template will be discarded/overridden when scheduling preferences are
	// specified.
	TotalReplicas int32

	// If set to true then already scheduled and running replicas may be moved to other clusters
	// in order to match current state to the specified preferences. Otherwise, if set to false,
	// up and running replicas will not be moved.
	// +optional
	Rebalance bool

	// A mapping between cluster names and preferences regarding a local workload object (dep, rs, .. ) in
	// these clusters.
	// "*" (if provided) applies to all clusters if an explicit mapping is not provided.
	// If omitted, clusters without explicit preferences should not have any replicas scheduled.
	// +optional
	Clusters map[string]ClusterPreferences
}

// Preferences regarding number of replicas assigned to a cluster workload object (dep, rs, ..) within
// a federated workload object.
type ClusterPreferences struct {
	// Minimum number of replicas that should be assigned to this cluster workload object. 0 by default.
	// +optional
	MinReplicas int64

	// Maximum number of replicas that should be assigned to this cluster workload object.
	// Unbounded if no value provided (default).
	// +optional
	MaxReplicas *int64

	// A number expressing the preference to put an additional replica to this cluster workload object.
	// 0 by default.
	Weight int64
}

// ReplicaSchedulingPreferenceStatus defines the observed state of ReplicaSchedulingPreference
type ReplicaSchedulingPreferenceStatus struct {
}

// Validate checks that an instance of ReplicaSchedulingPreference is well formed
func (ReplicaSchedulingPreferenceStrategy) Validate(ctx request.Context, obj runtime.Object) field.ErrorList {
	o := obj.(*federatedscheduling.ReplicaSchedulingPreference)
	log.Printf("Validating fields for ReplicaSchedulingPreference %s\n", o.Name)
	errors := field.ErrorList{}
	if o.Spec.TotalReplicas < 1 {
		errors = append(errors, field.Invalid(field.NewPath("replicaschedulingpreference.totalreplicas"), o.Spec.TotalReplicas, ""))
	}
	// perform validation here and add to errors using field.Invalid
	return errors
}

// DefaultingFunction sets default ReplicaSchedulingPreference field values
func (ReplicaSchedulingPreferenceSchemeFns) DefaultingFunction(o interface{}) {
	obj := o.(*ReplicaSchedulingPreference)
	// set default field values here
	log.Printf("Defaulting fields for ReplicaSchedulingPreference %s\n", obj.Name)
}
