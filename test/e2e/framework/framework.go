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

package framework

import (
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset_generated/clientset"
	"github.com/kubernetes-sigs/federation-v2/pkg/federatedtypes"
	kubeclientset "k8s.io/client-go/kubernetes"
	crclientset "k8s.io/cluster-registry/pkg/client/clientset_generated/clientset"

	"github.com/onsi/ginkgo"
)

// FederationFramework provides an interface to a test federation so
// that the implementation can vary without affecting tests.
type FederationFramework interface {
	BeforeEach()
	AfterEach()

	FedClient(userAgent string) fedclientset.Interface
	KubeClient(userAgent string) kubeclientset.Interface
	CrClient(userAgent string) crclientset.Interface

	ClusterClients(userAgent string) map[string]kubeclientset.Interface

	// Name of the namespace for the current test to target
	TestNamespaceName() string

	// Initialize and cleanup in-memory controller (useful for debugging)
	SetUpControllerFixture(kind string, adapterFactory federatedtypes.AdapterFactory)
}

// A framework needs to be instantiated before tests are executed to
// ensure registration of BeforeEach/AfterEach.  However, flags that
// dictate which framework should be used won't have been parsed yet.
// The workaround is using a wrapper that performs late-binding on the
// framework flavor.
type frameworkWrapper struct {
	impl     FederationFramework
	baseName string
}

func NewFederationFramework(baseName string) FederationFramework {
	f := &frameworkWrapper{
		baseName: baseName,
	}
	ginkgo.AfterEach(f.AfterEach)
	ginkgo.BeforeEach(f.BeforeEach)
	return f
}

func (f *frameworkWrapper) framework() FederationFramework {
	if f.impl == nil {
		if TestContext.TestManagedFederation {
			f.impl = NewManagedFramework(f.baseName)
		} else {
			f.impl = NewUnmanagedFramework(f.baseName)
		}
	}
	return f.impl
}

func (f *frameworkWrapper) BeforeEach() {
	f.framework().BeforeEach()
}

func (f *frameworkWrapper) AfterEach() {
	f.framework().AfterEach()
}

func (f *frameworkWrapper) FedClient(userAgent string) fedclientset.Interface {
	return f.framework().FedClient(userAgent)
}

func (f *frameworkWrapper) KubeClient(userAgent string) kubeclientset.Interface {
	return f.framework().KubeClient(userAgent)
}

func (f *frameworkWrapper) CrClient(userAgent string) crclientset.Interface {
	return f.framework().CrClient(userAgent)
}

func (f *frameworkWrapper) ClusterClients(userAgent string) map[string]kubeclientset.Interface {
	return f.framework().ClusterClients(userAgent)
}

func (f *frameworkWrapper) TestNamespaceName() string {
	return f.framework().TestNamespaceName()
}

func (f *frameworkWrapper) SetUpControllerFixture(kind string, adapterFactory federatedtypes.AdapterFactory) {
	f.framework().SetUpControllerFixture(kind, adapterFactory)
}
