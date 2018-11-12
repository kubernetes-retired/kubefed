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
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	"github.com/kubernetes-sigs/federation-v2/test/integration/framework"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeclientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	crclientset "k8s.io/cluster-registry/pkg/client/clientset/versioned"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type FederationFrameworkImpl interface {
	BeforeEach()
	AfterEach()

	ControllerConfig() *util.ControllerConfig

	Logger() common.TestLogger

	KubeConfig() *restclient.Config

	KubeClient(userAgent string) kubeclientset.Interface
	FedClient(userAgent string) fedclientset.Interface
	CrClient(userAgent string) crclientset.Interface

	ClusterConfigs(userAgent string) map[string]common.TestClusterConfig
	ClusterDynamicClients(apiResource *metav1.APIResource, userAgent string) map[string]common.TestCluster
	ClusterKubeClients(userAgent string) map[string]kubeclientset.Interface
	ClusterNames(userAgent string) []string

	FederationSystemNamespace() string

	// Name of the namespace for the current test to target
	TestNamespaceName() string

	// Setup a sync controller if necessary and return the fixture.
	// This is implemented commonly to support running a sync
	// controller for namespaces for tests that require it.
	SetUpSyncControllerFixture(typeConfig typeconfig.Interface) framework.TestFixture
}

// FederationFramework provides an interface to a test federation so
// that the implementation can vary without affecting tests.
type FederationFramework interface {
	FederationFrameworkImpl

	// Registering a fixture ensures it will be torn down after the
	// current test has executed.
	RegisterFixture(fixture framework.TestFixture)

	// Start a namespace sync controller fixture
	SetUpNamespaceSyncControllerFixture()
}

// A framework needs to be instantiated before tests are executed to
// ensure registration of BeforeEach/AfterEach.  However, flags that
// dictate which framework should be used won't have been parsed yet.
// The workaround is using a wrapper that performs late-binding on the
// framework flavor.
type frameworkWrapper struct {
	impl     FederationFrameworkImpl
	baseName string

	// Fixtures to cleanup after each test
	fixtures []framework.TestFixture
}

func NewFederationFramework(baseName string) FederationFramework {
	f := &frameworkWrapper{
		baseName: baseName,
		fixtures: []framework.TestFixture{},
	}
	AfterEach(f.AfterEach)
	BeforeEach(f.BeforeEach)
	return f
}

func (f *frameworkWrapper) framework() FederationFrameworkImpl {
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

	// TODO(font): Delete the namespace finalizer manually rather than
	// relying on the federated namespace sync controller. This would
	// remove the dependency of namespace removal on fixture teardown,
	// which allows the teardown to be moved outside the defer, but before
	// the DumpEventsInNamespace that may contain assertions that could
	// exit the function.
	logger := f.framework().Logger()
	for len(f.fixtures) > 0 {
		fixture := f.fixtures[0]
		fixture.TearDown(logger)
		f.fixtures = f.fixtures[1:]
	}
}

func (f *frameworkWrapper) ControllerConfig() *util.ControllerConfig {
	return f.framework().ControllerConfig()
}

func (f *frameworkWrapper) Logger() common.TestLogger {
	return f.framework().Logger()
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

func (f *frameworkWrapper) ClusterConfigs(userAgent string) map[string]common.TestClusterConfig {
	return f.framework().ClusterConfigs(userAgent)
}

func (f *frameworkWrapper) ClusterNames(userAgent string) []string {
	return f.framework().ClusterNames(userAgent)
}

func (f *frameworkWrapper) ClusterDynamicClients(apiResource *metav1.APIResource, userAgent string) map[string]common.TestCluster {
	return f.framework().ClusterDynamicClients(apiResource, userAgent)
}

func (f *frameworkWrapper) ClusterKubeClients(userAgent string) map[string]kubeclientset.Interface {
	return f.framework().ClusterKubeClients(userAgent)
}

func (f *frameworkWrapper) FederationSystemNamespace() string {
	return f.framework().FederationSystemNamespace()
}

func (f *frameworkWrapper) TestNamespaceName() string {
	return f.framework().TestNamespaceName()
}

func (f *frameworkWrapper) SetUpSyncControllerFixture(typeConfig typeconfig.Interface) framework.TestFixture {
	return f.framework().SetUpSyncControllerFixture(typeConfig)
}

func (f *frameworkWrapper) RegisterFixture(fixture framework.TestFixture) {
	if fixture != nil {
		f.fixtures = append(f.fixtures, fixture)
	}
}

func (f *frameworkWrapper) SetUpNamespaceSyncControllerFixture() {
	// When targeting a single namespace the namespace controller is not required.
	if TestContext.LimitedScope {
		return
	}
	// The namespace controller is required to ensure namespaces
	// are created as needed in member clusters in advance of
	// propagation of other namespaced types.
	namespaceTypeConfig := common.NamespaceTypeConfigOrDie(f.Logger())
	fixture := f.framework().SetUpSyncControllerFixture(namespaceTypeConfig)
	f.RegisterFixture(fixture)
}

func createTestNamespace(client kubeclientset.Interface, baseName string) string {
	By("Creating a namespace to execute the test in")
	namespaceName, err := createNamespace(client, baseName)
	Expect(err).NotTo(HaveOccurred())
	By(fmt.Sprintf("Created test namespace %s", namespaceName))
	return namespaceName
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
	if err := wait.PollImmediate(PollInterval, TestContext.SingleCallTimeout, func() (bool, error) {
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
