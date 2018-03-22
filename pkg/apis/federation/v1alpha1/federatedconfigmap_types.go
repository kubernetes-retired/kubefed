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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/marun/fnord/pkg/apis/federation"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FederatedConfigMap
// +k8s:openapi-gen=true
// +resource:path=federatedconfigmaps,strategy=FederatedConfigMapStrategy
type FederatedConfigMap struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FederatedConfigMapSpec   `json:"spec,omitempty"`
	Status FederatedConfigMapStatus `json:"status,omitempty"`
}

// FederatedConfigMapSpec defines the desired state of FederatedConfigMap
type FederatedConfigMapSpec struct {
	// Template to derive per-cluster configmap from
	Template corev1.ConfigMap `json:"template,omitempty"`
}

// FederatedConfigMapStatus defines the observed state of FederatedConfigMap
type FederatedConfigMapStatus struct {
}

// Validate checks that an instance of FederatedConfigMap is well formed
func (FederatedConfigMapStrategy) Validate(ctx request.Context, obj runtime.Object) field.ErrorList {
	o := obj.(*federation.FederatedConfigMap)
	log.Printf("Validating fields for FederatedConfigMap %s\n", o.Name)
	errors := field.ErrorList{}
	// perform validation here and add to errors using field.Invalid
	return errors
}

// DefaultingFunction sets default FederatedConfigMap field values
func (FederatedConfigMapSchemeFns) DefaultingFunction(o interface{}) {
	obj := o.(*FederatedConfigMap)
	// set default field values here
	log.Printf("Defaulting fields for FederatedConfigMap %s\n", obj.Name)
}
