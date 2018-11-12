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
	"github.com/kubernetes-sigs/federation-v2/test/e2e/framework/managed"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	crclientset "k8s.io/cluster-registry/pkg/client/clientset/versioned"
)

var (
	fedFixture *managed.FederationFixture
)

func SetUpManagedFederation() {
	if fedFixture != nil {
		return
	}
	fedFixture = managed.SetUpFederationFixture(NewE2ELogger(), 2)
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

	baseName string

	testNamespaceName string
}

func NewManagedFramework(baseName string) FederationFrameworkImpl {
	f := &ManagedFramework{
		logger:   NewE2ELogger(),
		baseName: baseName,
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
}

func (f *ManagedFramework) ControllerConfig() *util.ControllerConfig {
	return &util.ControllerConfig{
		FederationNamespaces: util.FederationNamespaces{
			FederationNamespace: fedFixture.SystemNamespace,
			ClusterNamespace:    fedFixture.SystemNamespace,
			TargetNamespace:     metav1.NamespaceAll,
		},
		KubeConfig:      fedFixture.KubeApi.NewConfig(f.logger),
		MinimizeLatency: true,
	}
}

func (f *ManagedFramework) Logger() common.TestLogger {
	return f.logger
}

func (f *ManagedFramework) KubeConfig() *restclient.Config {
	return fedFixture.KubeApi.NewConfig(f.logger)
}

func (f *ManagedFramework) FedClient(userAgent string) fedclientset.Interface {
	config := fedFixture.KubeApi.NewConfig(f.logger)
	restclient.AddUserAgent(config, userAgent)
	return fedclientset.NewForConfigOrDie(config)
}

func (f *ManagedFramework) KubeClient(userAgent string) kubeclientset.Interface {
	return fedFixture.KubeApi.NewClient(f.logger, userAgent)
}

func (f *ManagedFramework) CrClient(userAgent string) crclientset.Interface {
	config := fedFixture.KubeApi.NewConfig(f.logger)
	restclient.AddUserAgent(config, userAgent)
	return crclientset.NewForConfigOrDie(config)
}

func (f *ManagedFramework) ClusterNames(userAgent string) []string {
	return fedFixture.ClusterNames()
}

func (f *ManagedFramework) ClusterConfigs(userAgent string) map[string]common.TestClusterConfig {
	return fedFixture.ClusterConfigs(f.logger, userAgent)
}

func (f *ManagedFramework) ClusterDynamicClients(apiResource *metav1.APIResource, userAgent string) map[string]common.TestCluster {
	return fedFixture.ClusterDynamicClients(f.logger, apiResource, userAgent)
}

func (f *ManagedFramework) ClusterKubeClients(userAgent string) map[string]kubeclientset.Interface {
	return fedFixture.ClusterKubeClients(f.logger, userAgent)
}

func (f *ManagedFramework) FederationSystemNamespace() string {
	return fedFixture.SystemNamespace
}

func (f *ManagedFramework) TestNamespaceName() string {
	if f.testNamespaceName == "" {
		client := f.KubeClient(fmt.Sprintf("%s-create-namespace", f.baseName))
		f.testNamespaceName = createTestNamespace(client, f.baseName)
	}
	return f.testNamespaceName
}

func (f *ManagedFramework) SetUpSyncControllerFixture(typeConfig typeconfig.Interface) managed.TestFixture {
	return managed.NewSyncControllerFixture(f.logger, f.ControllerConfig(), typeConfig)
}
