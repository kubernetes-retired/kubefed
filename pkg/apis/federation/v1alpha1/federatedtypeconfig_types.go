
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

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/federation"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced

// FederatedTypeConfig
// +k8s:openapi-gen=true
// +resource:path=federatedtypeconfigs,strategy=FederatedTypeConfigStrategy
type FederatedTypeConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FederatedTypeConfigSpec   `json:"spec,omitempty"`
	Status FederatedTypeConfigStatus `json:"status,omitempty"`
}

// FederatedTypeConfigSpec defines the desired state of FederatedTypeConfig
type FederatedTypeConfigSpec struct {
}

// FederatedTypeConfigStatus defines the observed state of FederatedTypeConfig
type FederatedTypeConfigStatus struct {
}

// Validate checks that an instance of FederatedTypeConfig is well formed
func (FederatedTypeConfigStrategy) Validate(ctx request.Context, obj runtime.Object) field.ErrorList {
	o := obj.(*federation.FederatedTypeConfig)
	log.Printf("Validating fields for FederatedTypeConfig %s\n", o.Name)
	errors := field.ErrorList{}
	// perform validation here and add to errors using field.Invalid
	return errors
}

func (FederatedTypeConfigStrategy) NamespaceScoped() bool { return false }

func (FederatedTypeConfigStatusStrategy) NamespaceScoped() bool { return false }

// DefaultingFunction sets default FederatedTypeConfig field values
func (FederatedTypeConfigSchemeFns) DefaultingFunction(o interface{}) {
	obj := o.(*FederatedTypeConfig)
	// set default field values here
	log.Printf("Defaulting fields for FederatedTypeConfig %s\n", obj.Name)
}
