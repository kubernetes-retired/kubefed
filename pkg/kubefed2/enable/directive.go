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

package enable

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/common"
)

// EnableTypeDirectiveSpec defines the desired state of EnableTypeDirective.
type EnableTypeDirectiveSpec struct {
	// The API version of the target type.
	// +optional
	TargetVersion string `json:"targetVersion,omitempty"`

	// Which field of the target type determines whether federation
	// considers two resources to be equal.
	ComparisonField common.VersionComparisonField `json:"comparisonField"`

	// The name of the API group to use for generated federation types.
	// +optional
	FederationGroup string `json:"federationGroup,omitempty"`

	// The API version to use for generated federation types.
	// +optional
	FederationVersion string `json:"federationVersion,omitempty"`
}

// TODO(marun) This should become a proper API type and drive enabling
// type federation via a controller.  For now its only purpose is to
// enable loading of configuration from disk.
type EnableTypeDirective struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec EnableTypeDirectiveSpec `json:"spec,omitempty"`
}

func (ft *EnableTypeDirective) SetDefaults() {
	ft.Spec.FederationGroup = DefaultFederationGroup
	ft.Spec.FederationVersion = DefaultFederationVersion
}

func NewEnableTypeDirective() *EnableTypeDirective {
	ft := &EnableTypeDirective{}
	ft.SetDefaults()
	return ft
}
