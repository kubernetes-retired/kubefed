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
	"fmt"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeclientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	crclientset "k8s.io/cluster-registry/pkg/client/clientset/versioned"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// FederationFramework provides an interface to a test federation so
// that the implementation can vary without affecting tests.
type FederationFramework interface {
	BeforeEach()
	AfterEach()

	KubeConfig() *restclient.Config

	KubeClient(userAgent string) kubeclientset.Interface
	FedClient(userAgent string) fedclientset.Interface
	CrClient(userAgent string) crclientset.Interface

	ClusterDynamicClients(apiResource *metav1.APIResource, userAgent string) map[string]common.TestCluster
	ClusterKubeClients(userAgent string) map[string]kubeclientset.Interface

	// Name of the namespace for the current test to target
	TestNamespaceName() string

	// Initialize and cleanup in-memory controller (useful for debugging)
	SetUpControllerFixture(typeConfig typeconfig.Interface)

	SetUpServiceDNSControllerFixture()
}

// A framework needs to be instantiated before tests are executed to
// ensure registration of BeforeEach/AfterEach.  However, flags that
// dictate which framework should be used won't have been parsed yet.
// The workaround is using a wrapper that performs late-binding on the
// framework flavor.
type frameworkWrapper struct {
	impl              FederationFramework
	baseName          string
	testNamespaceName string
}

func NewFederationFramework(baseName string) FederationFramework {
	f := &frameworkWrapper{
		baseName: baseName,
	}
	AfterEach(f.AfterEach)
	BeforeEach(f.BeforeEach)
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

func (f *frameworkWrapper) KubeConfig() *restclient.Config {
	return f.framework().KubeConfig()
}

func (f *frameworkWrapper) KubeClient(userAgent string) kubeclientset.Interface {
	return f.framework().KubeClient(userAgent)
}

func (f *frameworkWrapper) FedClient(userAgent string) fedclientset.Interface {
	return f.framework().FedClient(userAgent)
}

func (f *frameworkWrapper) CrClient(userAgent string) crclientset.Interface {
	return f.framework().CrClient(userAgent)
}

func (f *frameworkWrapper) ClusterDynamicClients(apiResource *metav1.APIResource, userAgent string) map[string]common.TestCluster {
	return f.framework().ClusterDynamicClients(apiResource, userAgent)
}

func (f *frameworkWrapper) ClusterKubeClients(userAgent string) map[string]kubeclientset.Interface {
	return f.framework().ClusterKubeClients(userAgent)
}

func (f *frameworkWrapper) SetUpControllerFixture(typeConfig typeconfig.Interface) {
	f.framework().SetUpControllerFixture(typeConfig)
}

func (f *frameworkWrapper) TestNamespaceName() string {
	if f.testNamespaceName == "" {
		By("Creating a namespace to execute the test in")
		client := f.KubeClient(fmt.Sprintf("%s-create-namespace", f.baseName))
		namespaceName, err := createNamespace(client, f.baseName)
		Expect(err).NotTo(HaveOccurred())
		f.testNamespaceName = namespaceName
		By(fmt.Sprintf("Created test namespace %s", namespaceName))
	}
	return f.testNamespaceName
}

func createNamespace(client kubeclientset.Interface, baseName string) (string, error) {
	namespaceObj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("e2e-tests-%v-", baseName),
		},
	}
	// Be robust about making the namespace creation call.
	// TODO(marun) should all api calls be made 'robustly'?
	var namespaceName string
	if err := wait.PollImmediate(PollInterval, SingleCallTimeout, func() (bool, error) {
		namespace, err := client.Core().Namespaces().Create(namespaceObj)
		if err != nil {
			Logf("Unexpected error while creating namespace: %v", err)
			return false, nil
		}
		namespaceName = namespace.Name
		return true, nil
	}); err != nil {
		return "", err
	}
	return namespaceName, nil
}

func (f *frameworkWrapper) SetUpServiceDNSControllerFixture() {
	f.framework().SetUpServiceDNSControllerFixture()
}
