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
package fake

import (
	internalversion "github.com/marun/fnord/pkg/client/clientset_generated/internalclientset/typed/federation/internalversion"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeFederation struct {
	*testing.Fake
}

func (c *FakeFederation) FederatedClusters() internalversion.FederatedClusterInterface {
	return &FakeFederatedClusters{c}
}

func (c *FakeFederation) FederatedConfigMaps(namespace string) internalversion.FederatedConfigMapInterface {
	return &FakeFederatedConfigMaps{c, namespace}
}

func (c *FakeFederation) FederatedConfigMapOverrides(namespace string) internalversion.FederatedConfigMapOverrideInterface {
	return &FakeFederatedConfigMapOverrides{c, namespace}
}

func (c *FakeFederation) FederatedConfigMapPlacements(namespace string) internalversion.FederatedConfigMapPlacementInterface {
	return &FakeFederatedConfigMapPlacements{c, namespace}
}

func (c *FakeFederation) FederatedReplicaSets(namespace string) internalversion.FederatedReplicaSetInterface {
	return &FakeFederatedReplicaSets{c, namespace}
}

func (c *FakeFederation) FederatedReplicaSetOverrides(namespace string) internalversion.FederatedReplicaSetOverrideInterface {
	return &FakeFederatedReplicaSetOverrides{c, namespace}
}

func (c *FakeFederation) FederatedReplicaSetPlacements(namespace string) internalversion.FederatedReplicaSetPlacementInterface {
	return &FakeFederatedReplicaSetPlacements{c, namespace}
}

func (c *FakeFederation) FederatedSecrets(namespace string) internalversion.FederatedSecretInterface {
	return &FakeFederatedSecrets{c, namespace}
}

func (c *FakeFederation) FederatedSecretOverrides(namespace string) internalversion.FederatedSecretOverrideInterface {
	return &FakeFederatedSecretOverrides{c, namespace}
}

func (c *FakeFederation) FederatedSecretPlacements(namespace string) internalversion.FederatedSecretPlacementInterface {
	return &FakeFederatedSecretPlacements{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeFederation) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
