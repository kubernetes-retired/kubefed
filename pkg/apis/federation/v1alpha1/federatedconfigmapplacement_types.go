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

// FederatedConfigMapPlacement
// +k8s:openapi-gen=true
// +resource:path=federatedconfigmapplacements,strategy=FederatedConfigMapPlacementStrategy
type FederatedConfigMapPlacement struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FederatedConfigMapPlacementSpec   `json:"spec,omitempty"`
	Status FederatedConfigMapPlacementStatus `json:"status,omitempty"`
}

// FederatedConfigMapPlacementSpec defines the desired state of FederatedConfigMapPlacement
type FederatedConfigMapPlacementSpec struct {
	// Names of the clusters that a federated resource should exist in.
	ClusterNames []string `json:"clusternames,omitempty"`
}

// FederatedConfigMapPlacementStatus defines the observed state of FederatedConfigMapPlacement
type FederatedConfigMapPlacementStatus struct {
}

// Validate checks that an instance of FederatedConfigMapPlacement is well formed
func (FederatedConfigMapPlacementStrategy) Validate(ctx request.Context, obj runtime.Object) field.ErrorList {
	o := obj.(*federation.FederatedConfigMapPlacement)
	log.Printf("Validating fields for FederatedConfigMapPlacement %s\n", o.Name)
	errors := field.ErrorList{}
	// perform validation here and add to errors using field.Invalid
	return errors
}

// DefaultingFunction sets default FederatedConfigMapPlacement field values
func (FederatedConfigMapPlacementSchemeFns) DefaultingFunction(o interface{}) {
	obj := o.(*FederatedConfigMapPlacement)
	// set default field values here
	log.Printf("Defaulting fields for FederatedConfigMapPlacement %s\n", obj.Name)
}
