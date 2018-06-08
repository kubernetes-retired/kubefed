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
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/common"
)

// FederatedTypeConfigSpec defines the desired state of FederatedTypeConfig
type FederatedTypeConfigSpec struct {
	// The configuration of the target type.  Kind will be set to the
	// name of this resource regardles of the value provided.
	Target APIResource `json:"target"`
	// Whether or not the target resource is namespaced (all primitive
	// resources will share this).
	Namespaced bool `json:"namespaced"`
	// What field equality determines equality.
	ComparisonField common.VersionComparisonField `json:"comparisonField"`
	// Whether or not federation of the resource should be enabled.
	PropagationEnabled bool `json:"propagationEnabled"`
	// Configuration for the template resource.
	Template APIResource `json:"template"`
	// Configuration for the placement resource. If not provided, the
	// group and version will default to those provided for the
	// template resource.
	Placement APIResource `json:"placement"`
	// Configuration for the override resource. If not provided, the
	// group and version will default to those provided for the
	// template resource.
	// +optional
	Override *APIResource `json:"override,omitempty"`
	// The path to the field to override in the target type.  The last
	// entry in the path should be the name of the field in the override type.
	// +optional
	OverridePath []string `json:"overridePath,omitempty"`
}

// APIResource defines how to configure the dynamic client for an api resource.
type APIResource struct {
	// metav1.GroupVersion is not used since the json annotation of
	// the fields enforces them as mandatory.

	// Group of the resource.
	Group string `json:"group,omitempty"`
	// Version of the resource.
	Version string `json:"version,omitempty"`
	// Camel-cased singular name of the resource (e.g. ConfigMap)
	Kind string `json:"kind"`
	// Lower-cased plural name of the resource (e.g. configmaps).  If
	// not provided, it will be computed by lower-casing the kind and
	// suffixing an 's'.
	PluralName string `json:"pluralName,omitempty"`
}

// FederatedTypeConfigStatus defines the observed state of FederatedTypeConfig
type FederatedTypeConfigStatus struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced

// FederatedTypeConfig
// +k8s:openapi-gen=true
// +kubebuilder:resource:path=federatedtypeconfigs
type FederatedTypeConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FederatedTypeConfigSpec   `json:"spec,omitempty"`
	Status FederatedTypeConfigStatus `json:"status,omitempty"`
}

func SetFederatedTypeConfigDefaults(obj *FederatedTypeConfig) {
	// TODO(marun) will name always be populated?
	nameParts := strings.SplitN(obj.Name, ".", 2)
	templatePluralName := nameParts[0]
	setStringDefault(&obj.Spec.Target.PluralName, templatePluralName)
	if len(nameParts) > 1 {
		group := nameParts[1]
		setStringDefault(&obj.Spec.Target.Group, group)
	}
	setStringDefault(&obj.Spec.Template.PluralName, pluralName(obj.Spec.Template.Kind))
	setStringDefault(&obj.Spec.Placement.PluralName, pluralName(obj.Spec.Placement.Kind))
	setStringDefault(&obj.Spec.Placement.Group, obj.Spec.Template.Group)
	setStringDefault(&obj.Spec.Placement.Version, obj.Spec.Template.Version)
	if obj.Spec.Override != nil {
		setStringDefault(&obj.Spec.Override.PluralName, pluralName(obj.Spec.Override.Kind))
		setStringDefault(&obj.Spec.Override.Group, obj.Spec.Template.Group)
		setStringDefault(&obj.Spec.Override.Version, obj.Spec.Template.Version)
	}
}

// GetDefaultedString returns the value if provided, and otherwise
// returns the provided default.
func setStringDefault(value *string, defaultValue string) {
	if value == nil || len(*value) > 0 {
		return
	}
	*value = defaultValue
}

// PluralNameForKind naively computes the plural name from the kind by
// lowercasing and suffixing with 's'.
func pluralName(kind string) string {
	lowerKind := strings.ToLower(kind)
	if strings.HasSuffix(lowerKind, "overrides") {
		return fmt.Sprintf("%ses", lowerKind)
	}
	return fmt.Sprintf("%ss", lowerKind)
}

func (f *FederatedTypeConfig) GetTarget() metav1.APIResource {
	return apiResourceToMeta(f.Spec.Target, f.Spec.Namespaced)
}

func (f *FederatedTypeConfig) GetNamespaced() bool {
	return f.Spec.Namespaced
}

func (f *FederatedTypeConfig) GetComparisonField() common.VersionComparisonField {
	return f.Spec.ComparisonField
}

func (f *FederatedTypeConfig) GetPropagationEnabled() bool {
	return f.Spec.PropagationEnabled
}

func (f *FederatedTypeConfig) GetTemplate() metav1.APIResource {
	return apiResourceToMeta(f.Spec.Template, f.Spec.Namespaced)
}

func (f *FederatedTypeConfig) GetPlacement() metav1.APIResource {
	return apiResourceToMeta(f.Spec.Placement, f.Spec.Namespaced)
}

func (f *FederatedTypeConfig) GetOverride() *metav1.APIResource {
	if f.Spec.Override == nil {
		return nil
	}
	metaAPIResource := apiResourceToMeta(*f.Spec.Override, f.Spec.Namespaced)
	return &metaAPIResource
}

func (f *FederatedTypeConfig) GetOverridePath() []string {
	if len(f.Spec.OverridePath) == 0 {
		return nil
	}
	overridePath := make([]string, len(f.Spec.OverridePath))
	copy(overridePath, f.Spec.OverridePath)
	return overridePath
}

func apiResourceToMeta(apiResource APIResource, namespaced bool) metav1.APIResource {
	return metav1.APIResource{
		Group:      apiResource.Group,
		Version:    apiResource.Version,
		Kind:       apiResource.Kind,
		Name:       apiResource.PluralName,
		Namespaced: namespaced,
	}
}
