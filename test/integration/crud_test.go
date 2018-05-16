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
	"fmt"
	"strings"
	"testing"

	"github.com/pborman/uuid"

	"github.com/kubernetes-sigs/federation-v2/pkg/federatedtypes"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	"github.com/kubernetes-sigs/federation-v2/test/integration/framework"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
)

// TestCrud validates create/read/update/delete operations for federated types.
func TestCrud(t *testing.T) {
	tl := framework.NewIntegrationLogger(t)
	fedFixture := framework.SetUpFederationFixture(tl, 2)
	defer fedFixture.TearDown(tl)

	typeConfigs := federatedtypes.FederatedTypeConfigs()
	for templateKind, typeConfig := range typeConfigs {
		// TODO (font): integration tests for federated Namespace does not work
		// until k8s namespace controller is added.
		if federatedtypes.IsNamespaceKind(templateKind) {
			continue
		}

		t.Run(templateKind, func(t *testing.T) {
			tl := framework.NewIntegrationLogger(t)
			fixture, crudTester := initCrudTest(tl, fedFixture, typeConfig, templateKind)
			defer fixture.TearDown(tl)

			clusterNames := fedFixture.ClusterNames()
			template, placement, override, err := common.NewTestObjects(typeConfig, uuid.New(), clusterNames)
			if err != nil {
				tl.Fatalf("Error creating test objects: %v", err)
			}
			crudTester.CheckLifecycle(template, placement, override)
		})
	}
}

// initCrudTest initializes common elements of a crud test
func initCrudTest(tl common.TestLogger, fedFixture *framework.FederationFixture, typeConfig federatedtypes.FederatedTypeConfig, templateKind string) (
	*framework.ControllerFixture, *common.FederatedTypeCrudTester) {
	fedConfig := fedFixture.FedApi.NewConfig(tl)
	kubeConfig := fedFixture.KubeApi.NewConfig(tl)
	crConfig := fedFixture.CrApi.NewConfig(tl)
	fixture := framework.NewSyncControllerFixture(tl, typeConfig, fedConfig, kubeConfig, crConfig)

	userAgent := fmt.Sprintf("test-%s-crud", strings.ToLower(templateKind))
	rest.AddUserAgent(fedConfig, userAgent)
	rest.AddUserAgent(kubeConfig, userAgent)
	clusterClients := fedFixture.ClusterClients(tl, &typeConfig.Target, userAgent)
	crudTester, err := common.NewFederatedTypeCrudTester(tl, typeConfig, fedConfig, kubeConfig, clusterClients, framework.DefaultWaitInterval, wait.ForeverTestTimeout)
	if err != nil {
		tl.Fatalf("Error creating crudtester for %q: %v", templateKind, err)
	}

	return fixture, crudTester
}
