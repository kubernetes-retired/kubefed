/*
Copyright 2018 The Federation v2 Authors.

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
	"github.com/pborman/uuid"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/federation/typeconfig"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset_generated/clientset"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	"github.com/kubernetes-sigs/federation-v2/test/integration/framework"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	crclientset "k8s.io/cluster-registry/pkg/client/clientset_generated/clientset"
)

var (
	fedFixture *framework.FederationFixture
)

func SetUpManagedFederation() {
	if fedFixture != nil {
		return
	}
	fedFixture = framework.SetUpFederationFixture(NewE2ELogger(), 2)
}

func TearDownManagedFederation() {
	if fedFixture != nil {
		fedFixture.TearDown(NewE2ELogger())
		fedFixture = nil
	}
}

type ManagedFramework struct {
	// To make sure that this framework cleans up after itself, no matter what,
	// we install a Cleanup action before each test and clear it after.  If we
	// should abort, the AfterSuite hook should run all Cleanup actions.
	cleanupHandle CleanupActionHandle

	logger common.TestLogger

	// Fixtures to cleanup after each test
	fixtures []framework.TestFixture
}

func NewManagedFramework(baseName string) FederationFramework {
	f := &ManagedFramework{
		logger:   NewE2ELogger(),
		fixtures: []framework.TestFixture{},
	}
	return f
}

func (f *ManagedFramework) BeforeEach() {
	// The fact that we need this feels like a bug in ginkgo.
	// https://github.com/onsi/ginkgo/issues/222
	f.cleanupHandle = AddCleanupAction(f.AfterEach)
}

func (f *ManagedFramework) AfterEach() {
	RemoveCleanupAction(f.cleanupHandle)
	for len(f.fixtures) > 0 {
		fixture := f.fixtures[0]
		fixture.TearDown(f.logger)
		f.fixtures = f.fixtures[1:]
	}
}

func (f *ManagedFramework) FedConfig() *restclient.Config {
	return fedFixture.FedApi.NewConfig(f.logger)
}

func (f *ManagedFramework) KubeConfig() *restclient.Config {
	return fedFixture.KubeApi.NewConfig(f.logger)
}

func (f *ManagedFramework) FedClient(userAgent string) fedclientset.Interface {
	return fedFixture.FedApi.NewClient(f.logger, userAgent)
}

func (f *ManagedFramework) KubeClient(userAgent string) kubeclientset.Interface {
	return fedFixture.KubeApi.NewClient(f.logger, userAgent)
}

func (f *ManagedFramework) CrClient(userAgent string) crclientset.Interface {
	return fedFixture.CrApi.NewClient(f.logger, userAgent)
}

func (f *ManagedFramework) ClusterClients(apiResource *metav1.APIResource, userAgent string) map[string]common.TestCluster {
	return fedFixture.ClusterClients(f.logger, apiResource, userAgent)
}

func (f *ManagedFramework) TestNamespaceName() string {
	return uuid.New()
}

func (f *ManagedFramework) SetUpControllerFixture(typeConfig typeconfig.Interface) {
	// TODO(marun) check TestContext.InMemoryControllers before setting up controller fixture
	fedConfig := fedFixture.FedApi.NewConfig(f.logger)
	kubeConfig := fedFixture.KubeApi.NewConfig(f.logger)
	crConfig := fedFixture.CrApi.NewConfig(f.logger)
	fixture := framework.NewSyncControllerFixture(f.logger, typeConfig, fedConfig, kubeConfig, crConfig)
	f.fixtures = append(f.fixtures, fixture)
}
