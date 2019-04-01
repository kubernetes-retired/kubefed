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
)

// FederatedTypeConfigSpec defines the desired state of FederatedTypeConfig.
type FederatedTypeConfigSpec struct {
	// The configuration of the target type. If not set, the pluralName and
	// groupName fields will be set from the metadata.name of this resource. The
	// kind field must be set.
	Target APIResource `json:"target"`
	// Whether or not the target type is namespaced. The federation
	// types (FederatedType, Status) for the type will share this
	// characteristic.
	//
	// TODO(marun) Remove in favor of using the value from Target and
	// FederatedType (depending on context).
	Namespaced bool `json:"namespaced"`
	// Whether or not propagation to member clusters should be enabled.
	PropagationEnabled bool `json:"propagationEnabled"`
	// Configuration for the federated type that defines (via
	// template, placement and overrides fields) how the target type
	// should appear in multiple cluster.
	FederatedType APIResource `json:"federatedType"`
	// Configuration for the status type that holds information about which type
	// holds the status of the federated resource. If not provided, the group
	// and version will default to those provided for the federated type api
	// resource.
	// +optional
	Status *APIResource `json:"status,omitempty"`
	// Whether or not Status object should be populated.
	// +optional
	EnableStatus bool `json:"enableStatus,omitempty"`
}

// APIResource defines how to configure the dynamic client for an API resource.
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

// ControllerStatus defines the current state of the controller
type ControllerStatus string

const (
	// ControllerStatusRunning means controller is in "running" state
	ControllerStatusRunning ControllerStatus = "Running"
	// ControllerStatusNotRunning means controller is in "notrunning" state
	ControllerStatusNotRunning ControllerStatus = "NotRunning"
)

// FederatedTypeConfigStatus defines the observed state of FederatedTypeConfig
type FederatedTypeConfigStatus struct {
	// ObservedGeneration is the generation as observed by the controller consuming the FederatedTypeConfig.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// PropagationController tracks the status of the sync controller.
	// +optional
	PropagationController ControllerStatus `json:"propagationController,omitempty"`
	// StatusController tracks the status of the status controller.
	// +optional
	StatusController ControllerStatus `json:"statusController,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FederatedTypeConfig programs federation to know about a single API type - the
// "target type" - that a user wants to federate. For each target type, there is
// a corresponding FederatedType that has the following fields:
//
// - The "template" field specifies the basic definition of a federated resource
// - The "placement" field specifies the placement information for the federated
//   resource
// - The "overrides" field specifies how the target resource should vary across
//   clusters.
//
// +k8s:openapi-gen=true
// +kubebuilder:resource:path=federatedtypeconfigs
// +kubebuilder:subresource:status
type FederatedTypeConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FederatedTypeConfigSpec   `json:"spec,omitempty"`
	Status FederatedTypeConfigStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FederatedTypeConfigList contains a list of FederatedTypeConfig
type FederatedTypeConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FederatedTypeConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FederatedTypeConfig{}, &FederatedTypeConfigList{})
}

func SetFederatedTypeConfigDefaults(obj *FederatedTypeConfig) {
	// TODO(marun) will name always be populated?
	nameParts := strings.SplitN(obj.Name, ".", 2)
	targetPluralName := nameParts[0]
	setStringDefault(&obj.Spec.Target.PluralName, targetPluralName)
	if len(nameParts) > 1 {
		group := nameParts[1]
		setStringDefault(&obj.Spec.Target.Group, group)
	}
	setStringDefault(&obj.Spec.FederatedType.PluralName, PluralName(obj.Spec.FederatedType.Kind))
	if obj.Spec.Status != nil {
		setStringDefault(&obj.Spec.Status.PluralName, PluralName(obj.Spec.Status.Kind))
		setStringDefault(&obj.Spec.Status.Group, obj.Spec.FederatedType.Group)
		setStringDefault(&obj.Spec.Status.Version, obj.Spec.FederatedType.Version)
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

// PluralName computes the plural name from the kind by
// lowercasing and suffixing with 's' or `es`.
func PluralName(kind string) string {
	lowerKind := strings.ToLower(kind)
	if strings.HasSuffix(lowerKind, "s") || strings.HasSuffix(lowerKind, "x") ||
		strings.HasSuffix(lowerKind, "ch") || strings.HasSuffix(lowerKind, "sh") ||
		strings.HasSuffix(lowerKind, "z") || strings.HasSuffix(lowerKind, "o") {
		return fmt.Sprintf("%ses", lowerKind)
	}
	if strings.HasSuffix(lowerKind, "y") {
		lowerKind = strings.TrimSuffix(lowerKind, "y")
		return fmt.Sprintf("%sies", lowerKind)
	}
	return fmt.Sprintf("%ss", lowerKind)
}

func (f *FederatedTypeConfig) GetObjectMeta() metav1.ObjectMeta {
	return f.ObjectMeta
}

func (f *FederatedTypeConfig) GetTarget() metav1.APIResource {
	return apiResourceToMeta(f.Spec.Target, f.Spec.Namespaced)
}

func (f *FederatedTypeConfig) GetNamespaced() bool {
	return f.Spec.Namespaced
}

func (f *FederatedTypeConfig) GetPropagationEnabled() bool {
	return f.Spec.PropagationEnabled
}

func (f *FederatedTypeConfig) GetFederatedType() metav1.APIResource {
	return apiResourceToMeta(f.Spec.FederatedType, f.GetFederatedNamespaced())
}

func (f *FederatedTypeConfig) GetStatus() *metav1.APIResource {
	if f.Spec.Status == nil {
		return nil
	}
	metaAPIResource := apiResourceToMeta(*f.Spec.Status, f.Spec.Namespaced)
	return &metaAPIResource
}

func (f *FederatedTypeConfig) GetEnableStatus() bool {
	return f.Spec.EnableStatus
}

// TODO(marun) Remove in favor of using 'true' for namespaces and the
// value from target otherwise.
func (f *FederatedTypeConfig) GetFederatedNamespaced() bool {
	// Special-case the scope of federated namespace since it will
	// hopefully be the only instance of the scope of a federated
	// type differing from the scope of its target.

	// TODO(marun) Use the constant in pkg/controller/util
	if f.Name == "namespaces" {
		// Namespace placement is namespaced to allow the control
		// plane to run with only namespace-scoped permissions.
		return true
	}
	return f.Spec.Namespaced
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
