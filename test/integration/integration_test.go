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

package integration

import (
	"testing"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	"github.com/kubernetes-sigs/federation-v2/test/integration/framework"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var FedFixture *framework.FederationFixture
var TypeConfigs []typeconfig.Interface

// TestIntegration provides a common setup and teardown for all the integration tests.
func TestIntegration(t *testing.T) {
	tl := framework.NewIntegrationLogger(t)
	FedFixture = framework.SetUpFederationFixture(tl, 2)
	defer FedFixture.TearDown(tl)

	var err error
	TypeConfigs, err = common.FederatedTypeConfigs()
	if err != nil {
		t.Fatalf("Error loading type configs: %v", err)
	}

	// Find the namespace typeconfig to be able to start its sync
	// controller.  This is safe for integration tests since the
	// namespace sync controller can't be CRUD tested due to the lack
	// of a kube namespace controller.
	//
	// TODO(marun) Remove the crud testing from integration since it
	// is duplicated by e2e.
	var namespaceTypeConfig typeconfig.Interface
	for _, typeConfig := range TypeConfigs {
		if typeConfig.GetTarget().Kind == util.NamespaceKind {
			namespaceTypeConfig = typeConfig
			break
		}
	}
	if namespaceTypeConfig == nil {
		t.Fatal("Unable to find namespace type config")
	}
	kubeConfig := FedFixture.KubeApi.NewConfig(tl)
	namespaceSyncFixture := framework.NewSyncControllerFixture(tl, namespaceTypeConfig, kubeConfig, FedFixture.SystemNamespace, FedFixture.SystemNamespace, metav1.NamespaceAll)
	defer namespaceSyncFixture.TearDown(tl)

	t.Run("Parallel-Integration-Test-Group", func(t *testing.T) {
		t.Run("TestCrud", TestCrud)
		t.Run("TestReplicaSchedulingPreference", TestReplicaSchedulingPreference)
		t.Run("TestServiceDNS", TestServiceDNS)
	})
}
