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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/marun/fnord/pkg/apis/federation"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced

// FederatedNamespace
// +k8s:openapi-gen=true
// +resource:path=federatednamespaces,strategy=FederatedNamespaceStrategy
type FederatedNamespace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FederatedNamespaceSpec   `json:"spec,omitempty"`
	Status FederatedNamespaceStatus `json:"status,omitempty"`
}

// FederatedNamespaceSpec defines the desired state of FederatedNamespace
type FederatedNamespaceSpec struct {
	// Template to derive per-cluster namespace from
	Template corev1.Namespace `json:"status,omitempty"`
}

// FederatedNamespaceStatus defines the observed state of FederatedNamespace
type FederatedNamespaceStatus struct {
}

// Validate checks that an instance of FederatedNamespace is well formed
func (FederatedNamespaceStrategy) Validate(ctx request.Context, obj runtime.Object) field.ErrorList {
	o := obj.(*federation.FederatedNamespace)
	log.Printf("Validating fields for FederatedNamespace %s\n", o.Name)
	errors := field.ErrorList{}
	// perform validation here and add to errors using field.Invalid
	return errors
}

func (FederatedNamespaceStrategy) NamespaceScoped() bool { return false }

func (FederatedNamespaceStatusStrategy) NamespaceScoped() bool { return false }

// DefaultingFunction sets default FederatedNamespace field values
func (FederatedNamespaceSchemeFns) DefaultingFunction(o interface{}) {
	obj := o.(*FederatedNamespace)
	// set default field values here
	log.Printf("Defaulting fields for FederatedNamespace %s\n", obj.Name)
}
