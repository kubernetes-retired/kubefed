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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/endpoints/request"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/federation"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FederatedJobOverride
// +k8s:openapi-gen=true
// +resource:path=federatedjoboverrides,strategy=FederatedJobOverrideStrategy
type FederatedJobOverride struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FederatedJobOverrideSpec   `json:"spec,omitempty"`
	Status FederatedJobOverrideStatus `json:"status,omitempty"`
}

// FederatedJobOverrideSpec defines the desired state of FederatedJobOverride
type FederatedJobOverrideSpec struct {
	Overrides []FederatedJobClusterOverride `json:"overrides,omitempty"`
}

// FederatedJobClusterOverride defines the overrides for a named cluster
type FederatedJobClusterOverride struct {
	// TODO(marun) Need to ensure that a cluster name only appears
	// once.  Why can't maps be used so this validation is automatic?
	ClusterName string `json:"clustername,omitempty"`
	Parallelism *int32 `json:"parallelism,omitempty"`
}

// FederatedJobOverrideStatus defines the observed state of FederatedJobOverride
type FederatedJobOverrideStatus struct {
}

// Validate checks that an instance of FederatedJobOverride is well formed
func (FederatedJobOverrideStrategy) Validate(ctx request.Context, obj runtime.Object) field.ErrorList {
	o := obj.(*federation.FederatedJobOverride)
	log.Printf("Validating fields for FederatedJobOverride %s\n", o.Name)
	errors := field.ErrorList{}
	// perform validation here and add to errors using field.Invalid
	return errors
}

// DefaultingFunction sets default FederatedJobOverride field values
func (FederatedJobOverrideSchemeFns) DefaultingFunction(o interface{}) {
	obj := o.(*FederatedJobOverride)
	// set default field values here
	log.Printf("Defaulting fields for FederatedJobOverride %s\n", obj.Name)
}
