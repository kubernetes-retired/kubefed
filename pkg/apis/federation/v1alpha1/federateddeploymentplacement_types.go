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

// FederatedDeploymentPlacement
// +k8s:openapi-gen=true
// +resource:path=federateddeploymentplacements,strategy=FederatedDeploymentPlacementStrategy
type FederatedDeploymentPlacement struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FederatedDeploymentPlacementSpec   `json:"spec,omitempty"`
	Status FederatedDeploymentPlacementStatus `json:"status,omitempty"`
}

// FederatedDeploymentPlacementSpec defines the desired state of FederatedDeploymentPlacement
type FederatedDeploymentPlacementSpec struct {
	ClusterNames []string `json:"clusternames,omitempty"`
}

// FederatedDeploymentPlacementStatus defines the observed state of FederatedDeploymentPlacement
type FederatedDeploymentPlacementStatus struct {
}

// Validate checks that an instance of FederatedDeploymentPlacement is well formed
func (FederatedDeploymentPlacementStrategy) Validate(ctx request.Context, obj runtime.Object) field.ErrorList {
	o := obj.(*federation.FederatedDeploymentPlacement)
	log.Printf("Validating fields for FederatedDeploymentPlacement %s\n", o.Name)
	errors := field.ErrorList{}
	// perform validation here and add to errors using field.Invalid
	return errors
}

// DefaultingFunction sets default FederatedDeploymentPlacement field values
func (FederatedDeploymentPlacementSchemeFns) DefaultingFunction(o interface{}) {
	obj := o.(*FederatedDeploymentPlacement)
	// set default field values here
	log.Printf("Defaulting fields for FederatedDeploymentPlacement %s\n", obj.Name)
}
