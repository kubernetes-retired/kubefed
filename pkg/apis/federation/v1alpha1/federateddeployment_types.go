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

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/endpoints/request"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/federation"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FederatedDeployment
// +k8s:openapi-gen=true
// +resource:path=federateddeployments,strategy=FederatedDeploymentStrategy
type FederatedDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FederatedDeploymentSpec   `json:"spec,omitempty"`
	Status FederatedDeploymentStatus `json:"status,omitempty"`
}

// FederatedDeploymentSpec defines the desired state of FederatedDeployment
type FederatedDeploymentSpec struct {
	Template appsv1.Deployment `json:"template,omitempty"`
}

// FederatedDeploymentStatus defines the observed state of FederatedDeployment
type FederatedDeploymentStatus struct {
}

// Validate checks that an instance of FederatedDeployment is well formed
func (FederatedDeploymentStrategy) Validate(ctx request.Context, obj runtime.Object) field.ErrorList {
	o := obj.(*federation.FederatedDeployment)
	log.Printf("Validating fields for FederatedDeployment %s\n", o.Name)
	errors := field.ErrorList{}
	// perform validation here and add to errors using field.Invalid
	return errors
}

// DefaultingFunction sets default FederatedDeployment field values
func (FederatedDeploymentSchemeFns) DefaultingFunction(o interface{}) {
	obj := o.(*FederatedDeployment)
	// set default field values here
	log.Printf("Defaulting fields for FederatedDeployment %s\n", obj.Name)
}
