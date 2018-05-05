/*
Copyright 2017 The Federation v2 Authors.

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

package federatedtypes

import (
	"fmt"

	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type FederationAPIResource struct {
	metav1.APIResource

	// To support testing without aggregation, the configuration for a
	// federation resource needs to indicate whether kube api should
	// be used instead of the federation api. Namespaces and CRDs
	// should be configured to use the kube api.
	UseKubeAPI bool
}

// TODO(marun) This should be an api type instead of being statically defined.
type FederatedTypeConfig struct {
	ComparisonType util.VersionCompareType
	Namespaced     bool
	Template       FederationAPIResource
	Placement      FederationAPIResource
	Override       *FederationAPIResource
	OverridePath   []string
	Target         metav1.APIResource
}

var typeRegistry = make(map[string]FederatedTypeConfig)

// RegisterFederatedTypeConfig ensures that configuration for the given template kind will be returned by the Propagations method.
func RegisterFederatedTypeConfig(typeConfig FederatedTypeConfig) {
	templateKind := typeConfig.Template.Kind
	_, ok := typeRegistry[templateKind]
	if ok {
		// TODO(marun) Is panicking ok given that this is part of a type-registration mechanism?
		panic(fmt.Sprintf("Configuration for %q has already been registered", templateKind))
	}
	typeRegistry[templateKind] = typeConfig
}

// FederatedTypeConfigs returns a mapping of kind
// (e.g. "FederatedSecret") to its configuration.
func FederatedTypeConfigs() map[string]FederatedTypeConfig {
	// TODO copy to avoid accidental mutation
	result := make(map[string]FederatedTypeConfig)
	for key, value := range typeRegistry {
		result[key] = value
	}
	return result
}
