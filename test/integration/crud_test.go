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

package integration

import (
	"fmt"
	"testing"

	"github.com/pborman/uuid"

	"github.com/kubernetes-sigs/federation-v2/pkg/federatedtypes"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	"github.com/kubernetes-sigs/federation-v2/test/integration/framework"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
)

// TestCrud validates create/read/update/delete operations for federated types.
func TestCrud(t *testing.T) {
	tl := framework.NewIntegrationLogger(t)
	fedFixture := framework.SetUpFederationFixture(tl, 2)
	defer fedFixture.TearDown(tl)

	fedTypeConfigs := federatedtypes.FederatedTypeConfigs()
	for kind, fedTypeConfig := range fedTypeConfigs {
		// TODO (font): integration tests for federated Namespace does not work
		// until k8s namespace controller is added.
		if kind == federatedtypes.NamespaceKind {
			continue
		}

		t.Run(kind, func(t *testing.T) {
			tl := framework.NewIntegrationLogger(t)
			fixture, crudTester, template, placement, override := initCrudTest(tl, fedFixture, fedTypeConfig.AdapterFactory, kind)
			defer fixture.TearDown(tl)

			crudTester.CheckLifecycle(template, placement, override)
		})
	}
}

// initCrudTest initializes common elements of a crud test
func initCrudTest(tl common.TestLogger, fedFixture *framework.FederationFixture, adapterFactory federatedtypes.AdapterFactory, kind string) (
	*framework.ControllerFixture, *common.FederatedTypeCrudTester, pkgruntime.Object, pkgruntime.Object, pkgruntime.Object) {
	// TODO(marun) stop requiring user agent when creating new config or clients
	fedConfig := fedFixture.FedApi.NewConfig(tl)
	kubeConfig := fedFixture.KubeApi.NewConfig(tl)
	crConfig := fedFixture.CrApi.NewConfig(tl)
	fixture := framework.NewSyncControllerFixture(tl, kind, adapterFactory, fedConfig, kubeConfig, crConfig)

	userAgent := fmt.Sprintf("crud-test-%s", kind)

	client := fedFixture.FedApi.NewClient(tl, userAgent)
	adapter := adapterFactory(client)

	clusterClients := fedFixture.ClusterClients(tl, userAgent)
	crudTester := common.NewFederatedTypeCrudTester(tl, adapter, clusterClients, framework.DefaultWaitInterval, wait.ForeverTestTimeout)

	clusterNames := fedFixture.ClusterNames()
	template, placement, override := federatedtypes.NewTestObjects(kind, uuid.New(), clusterNames)

	return fixture, crudTester, template, placement, override
}
