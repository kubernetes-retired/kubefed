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
	v1alpha1 "github.com/marun/fnord/pkg/client/clientset_generated/clientset/typed/federation/v1alpha1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeFederationV1alpha1 struct {
	*testing.Fake
}

func (c *FakeFederationV1alpha1) FederatedClusters() v1alpha1.FederatedClusterInterface {
	return &FakeFederatedClusters{c}
}

func (c *FakeFederationV1alpha1) FederatedReplicaSets(namespace string) v1alpha1.FederatedReplicaSetInterface {
	return &FakeFederatedReplicaSets{c, namespace}
}

func (c *FakeFederationV1alpha1) FederatedReplicaSetOverrides(namespace string) v1alpha1.FederatedReplicaSetOverrideInterface {
	return &FakeFederatedReplicaSetOverrides{c, namespace}
}

func (c *FakeFederationV1alpha1) FederatedSecrets(namespace string) v1alpha1.FederatedSecretInterface {
	return &FakeFederatedSecrets{c, namespace}
}

func (c *FakeFederationV1alpha1) FederatedSecretOverrides(namespace string) v1alpha1.FederatedSecretOverrideInterface {
	return &FakeFederatedSecretOverrides{c, namespace}
}

func (c *FakeFederationV1alpha1) FederatedSecretPlacements(namespace string) v1alpha1.FederatedSecretPlacementInterface {
	return &FakeFederatedSecretPlacements{c, namespace}
}

func (c *FakeFederationV1alpha1) FederationPlacements(namespace string) v1alpha1.FederationPlacementInterface {
	return &FakeFederationPlacements{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeFederationV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
