/*
Copyright 2017 The Kubernetes Authors.

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

	fedclientset "github.com/marun/fnord/pkg/client/clientset_generated/clientset"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// FederatedTypeConfig configures federation of a target type
type FederatedTypeConfig struct {
	Kind              string
	ControllerName    string
	RequiredResources []schema.GroupVersionResource
	AdapterFactory    AdapterFactory
}

var typeRegistry = make(map[string]FederatedTypeConfig)

// AdapterFactory defines the function signature for factory methods
// that create instances of a FederatedTypeAdapter.  Such methods
// should be registered with RegisterAdapterFactory to ensure the type
// adapter is discoverable.
type AdapterFactory func(client fedclientset.Interface) FederatedTypeAdapter

// RegisterFederatedType ensures that configuration for the given kind will be returned by the FederatedTypes method.
func RegisterFederatedType(kind string, factory AdapterFactory) {
	_, ok := typeRegistry[kind]
	if ok {
		// TODO Is panicking ok given that this is part of a type-registration mechanism
		panic(fmt.Sprintf("Type %q has already been registered", kind))
	}
	typeRegistry[kind] = FederatedTypeConfig{
		Kind:           kind,
		AdapterFactory: factory,
	}
}

// FederatedTypeConfigs returns a mapping of kind (e.g. "FederatedSecret") to the
// type information required to configure its federation.
func FederatedTypeConfigs() map[string]FederatedTypeConfig {
	// TODO copy to avoid accidental mutation
	result := make(map[string]FederatedTypeConfig)
	for key, value := range typeRegistry {
		result[key] = value
	}
	return result
}
