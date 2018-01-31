
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

// FederatedSecretOverrides
// +k8s:openapi-gen=true
// +resource:path=federatedsecretoverrides,strategy=FederatedSecretOverridesStrategy
type FederatedSecretOverrides struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FederatedSecretOverridesSpec   `json:"spec,omitempty"`
	Status FederatedSecretOverridesStatus `json:"status,omitempty"`
}

// FederatedSecretOverridesSpec defines the desired state of FederatedSecretOverrides
type FederatedSecretOverridesSpec struct {
}

// FederatedSecretOverridesStatus defines the observed state of FederatedSecretOverrides
type FederatedSecretOverridesStatus struct {
}

// Validate checks that an instance of FederatedSecretOverrides is well formed
func (FederatedSecretOverridesStrategy) Validate(ctx request.Context, obj runtime.Object) field.ErrorList {
	o := obj.(*federation.FederatedSecretOverrides)
	log.Printf("Validating fields for FederatedSecretOverrides %s\n", o.Name)
	errors := field.ErrorList{}
	// perform validation here and add to errors using field.Invalid
	return errors
}

// DefaultingFunction sets default FederatedSecretOverrides field values
func (FederatedSecretOverridesSchemeFns) DefaultingFunction(o interface{}) {
	obj := o.(*FederatedSecretOverrides)
	// set default field values here
	log.Printf("Defaulting fields for FederatedSecretOverrides %s\n", obj.Name)
}
