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

// PropagatedVersion
// +k8s:openapi-gen=true
// +resource:path=propagatedversions,strategy=PropagatedVersionStrategy
type PropagatedVersion struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PropagatedVersionSpec   `json:"spec,omitempty"`
	Status PropagatedVersionStatus `json:"status,omitempty"`
}

// PropagatedVersionSpec defines the desired state of PropagatedVersion
type PropagatedVersionSpec struct {
}

// PropagatedVersionStatus defines the observed state of PropagatedVersion
type PropagatedVersionStatus struct {
	TemplateVersion string                 `json:"templateversion,omitempty"`
	OverrideVersion string                 `json:"overrideversion,omitempty"`
	ClusterVersions []ClusterObjectVersion `json:"clusterversions,omitempty"`
}

type ClusterObjectVersion struct {
	ClusterName string `json:"clustername,omitempty"`
	Version     string `json:"version,omitempty"`
}

// Validate checks that an instance of PropagatedVersion is well formed
func (PropagatedVersionStrategy) Validate(ctx request.Context, obj runtime.Object) field.ErrorList {
	o := obj.(*federation.PropagatedVersion)
	log.Printf("Validating fields for PropagatedVersion %s\n", o.Name)
	errors := field.ErrorList{}
	// perform validation here and add to errors using field.Invalid
	return errors
}

// DefaultingFunction sets default PropagatedVersion field values
func (PropagatedVersionSchemeFns) DefaultingFunction(o interface{}) {
	obj := o.(*PropagatedVersion)
	// set default field values here
	log.Printf("Defaulting fields for PropagatedVersion %s\n", obj.Name)
}
