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
	"strings"
	"testing"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	"github.com/kubernetes-sigs/federation-v2/test/integration/framework"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
)

// TestCrud validates create/read/update/delete operations for federated types.
var TestCrud = func(t *testing.T) {
	t.Parallel()

	for _, typeConfig := range TypeConfigs {
		templateKind := typeConfig.GetTemplate().Kind

		// TODO (font): integration tests for federated Namespace does not work
		// until k8s namespace controller is added.
		if templateKind == util.NamespaceKind {
			continue
		}

		t.Run(templateKind, func(t *testing.T) {
			tl := framework.NewIntegrationLogger(t)
			fixture, crudTester := initCrudTest(tl, FedFixture, typeConfig, templateKind)
			defer fixture.TearDown(tl)

			baseName := fmt.Sprintf("crud-%s-", strings.ToLower(templateKind))
			kubeClient := FedFixture.KubeApi.NewClient(tl, baseName)
			testNamespace := framework.CreateTestNamespace(tl, kubeClient, baseName)
			clusterNames := FedFixture.ClusterNames()
			template, placement, override, err := common.NewTestObjects(typeConfig, testNamespace, clusterNames)
			if err != nil {
				tl.Fatalf("Error creating test objects: %v", err)
			}
			crudTester.CheckLifecycle(template, placement, override)
		})
	}
}

// initCrudTest initializes common elements of a crud test
func initCrudTest(tl common.TestLogger, fedFixture *framework.FederationFixture, typeConfig typeconfig.Interface, templateKind string) (
	*framework.ControllerFixture, *common.FederatedTypeCrudTester) {
	kubeConfig := fedFixture.KubeApi.NewConfig(tl)
	fixture := framework.NewSyncControllerFixture(tl, typeConfig, kubeConfig, fedFixture.SystemNamespace)

	userAgent := fmt.Sprintf("test-%s-crud", strings.ToLower(templateKind))
	rest.AddUserAgent(kubeConfig, userAgent)
	targetAPIResource := typeConfig.GetTarget()
	clusterClients := fedFixture.ClusterDynamicClients(tl, &targetAPIResource, userAgent)
	crudTester, err := common.NewFederatedTypeCrudTester(tl, typeConfig, kubeConfig, clusterClients, framework.DefaultWaitInterval, wait.ForeverTestTimeout)
	if err != nil {
		tl.Fatalf("Error creating crudtester for %q: %v", templateKind, err)
	}

	return fixture, crudTester
}
