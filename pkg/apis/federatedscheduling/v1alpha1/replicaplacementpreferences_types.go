
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
	"log"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/marun/federation-v2/pkg/apis/federatedscheduling"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ReplicaPlacementPreferences
// +k8s:openapi-gen=true
// +resource:path=replicaplacementpreferences,strategy=ReplicaPlacementPreferencesStrategy
type ReplicaPlacementPreferences struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ReplicaPlacementPreferencesSpec   `json:"spec,omitempty"`
	Status ReplicaPlacementPreferencesStatus `json:"status,omitempty"`
}

// ReplicaPlacementPreferencesSpec defines the desired state of ReplicaPlacementPreferences
type ReplicaPlacementPreferencesSpec struct {
}

// ReplicaPlacementPreferencesStatus defines the observed state of ReplicaPlacementPreferences
type ReplicaPlacementPreferencesStatus struct {
}

// Validate checks that an instance of ReplicaPlacementPreferences is well formed
func (ReplicaPlacementPreferencesStrategy) Validate(ctx request.Context, obj runtime.Object) field.ErrorList {
	o := obj.(*federatedscheduling.ReplicaPlacementPreferences)
	log.Printf("Validating fields for ReplicaPlacementPreferences %s\n", o.Name)
	errors := field.ErrorList{}
	// perform validation here and add to errors using field.Invalid
	return errors
}

// DefaultingFunction sets default ReplicaPlacementPreferences field values
func (ReplicaPlacementPreferencesSchemeFns) DefaultingFunction(o interface{}) {
	obj := o.(*ReplicaPlacementPreferences)
	// set default field values here
	log.Printf("Defaulting fields for ReplicaPlacementPreferences %s\n", obj.Name)
}
