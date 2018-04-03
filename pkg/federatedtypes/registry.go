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

	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset_generated/clientset"
)

// FederatedTypeConfig configures propagation of a federated type
type FederatedTypeConfig struct {
	Kind           string
	ControllerName string
	AdapterFactory AdapterFactory
}

var typeRegistry = make(map[string]FederatedTypeConfig)

// AdapterFactory defines the function signature for factory methods
// that create instances of a FederatedTypeAdapter.  Such methods
// should be registered with RegisterAdapterFactory to ensure the type
// adapter is discoverable.
type AdapterFactory func(client fedclientset.Interface) FederatedTypeAdapter

// RegisterFederatedTypeConfig ensures that configuration for the given kind will be returned by the Propagations method.
func RegisterFederatedTypeConfig(kind string, factory AdapterFactory) {
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
