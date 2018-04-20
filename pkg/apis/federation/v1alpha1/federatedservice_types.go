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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/federation"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FederatedService
// +k8s:openapi-gen=true
// +resource:path=federatedservices,strategy=FederatedServiceStrategy
type FederatedService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FederatedServiceSpec   `json:"spec,omitempty"`
	Status FederatedServiceStatus `json:"status,omitempty"`
}

// FederatedServiceSpec defines the desired state of FederatedService
type FederatedServiceSpec struct {
	// Template to derive per-cluster service from
	Template corev1.Service `json:"template,omitempty"`
}

// FederatedServiceStatus defines the observed state of FederatedService
type FederatedServiceStatus struct {
}

// Validate checks that an instance of FederatedService is well formed
func (FederatedServiceStrategy) Validate(ctx request.Context, obj runtime.Object) field.ErrorList {
	o := obj.(*federation.FederatedService)
	log.Printf("Validating fields for FederatedService %s\n", o.Name)
	errors := field.ErrorList{}
	// perform validation here and add to errors using field.Invalid
	return errors
}

// DefaultingFunction sets default FederatedService field values
func (FederatedServiceSchemeFns) DefaultingFunction(o interface{}) {
	obj := o.(*FederatedService)
	// set default field values here
	log.Printf("Defaulting fields for FederatedService %s\n", obj.Name)
}
