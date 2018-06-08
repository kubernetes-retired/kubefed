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

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/federation"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced

// FederatedNamespacePlacement
// +k8s:openapi-gen=true
// +resource:path=federatednamespaceplacements,strategy=FederatedNamespacePlacementStrategy
type FederatedNamespacePlacement struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FederatedNamespacePlacementSpec   `json:"spec,omitempty"`
	Status FederatedNamespacePlacementStatus `json:"status,omitempty"`
}

// FederatedNamespacePlacementSpec defines the desired state of FederatedNamespacePlacement
type FederatedNamespacePlacementSpec struct {
	// Names of the clusters that a federated resource should exist in.
	ClusterNames []string `json:"clusternames,omitempty"`
}

// FederatedNamespacePlacementStatus defines the observed state of FederatedNamespacePlacement
type FederatedNamespacePlacementStatus struct {
}

// Validate checks that an instance of FederatedNamespacePlacement is well formed
func (FederatedNamespacePlacementStrategy) Validate(ctx request.Context, obj runtime.Object) field.ErrorList {
	o := obj.(*federation.FederatedNamespacePlacement)
	log.Printf("Validating fields for FederatedNamespacePlacement %s\n", o.Name)
	errors := field.ErrorList{}
	// perform validation here and add to errors using field.Invalid
	return errors
}

func (FederatedNamespacePlacementStrategy) NamespaceScoped() bool { return false }

func (FederatedNamespacePlacementStatusStrategy) NamespaceScoped() bool { return false }

// DefaultingFunction sets default FederatedNamespacePlacement field values
func (FederatedNamespacePlacementSchemeFns) DefaultingFunction(o interface{}) {
	obj := o.(*FederatedNamespacePlacement)
	// set default field values here
	log.Printf("Defaulting fields for FederatedNamespacePlacement %s\n", obj.Name)
}
