
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

	"github.com/marun/fnord/pkg/apis/federation"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FederationPlacement
// +k8s:openapi-gen=true
// +resource:path=federationplacements,strategy=FederationPlacementStrategy
type FederationPlacement struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FederationPlacementSpec   `json:"spec,omitempty"`
	Status FederationPlacementStatus `json:"status,omitempty"`
}

// FederationPlacementSpec defines the desired state of FederationPlacement
type FederationPlacementSpec struct {
	ResourceSelector metav1.LabelSelector
	// TODO(marun) is gt/lt required as per https://github.com/kubernetes/federation/blob/master/apis/federation/v1beta1/types.go#L130
	ClusterSelector metav1.LabelSelector
}

// FederationPlacementStatus defines the observed state of FederationPlacement
type FederationPlacementStatus struct {
}

// Validate checks that an instance of FederationPlacement is well formed
func (FederationPlacementStrategy) Validate(ctx request.Context, obj runtime.Object) field.ErrorList {
	o := obj.(*federation.FederationPlacement)
	log.Printf("Validating fields for FederationPlacement %s\n", o.Name)
	errors := field.ErrorList{}
	// perform validation here and add to errors using field.Invalid
	return errors
}

// DefaultingFunction sets default FederationPlacement field values
func (FederationPlacementSchemeFns) DefaultingFunction(o interface{}) {
	obj := o.(*FederationPlacement)
	// set default field values here
	log.Printf("Defaulting fields for FederationPlacement %s\n", obj.Name)
}
