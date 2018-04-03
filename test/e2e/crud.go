/*
Copyright 2017 The Kubernetes Authors.

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

package e2e

import (
	"fmt"

	"github.com/kubernetes-sigs/federation-v2/pkg/federatedtypes"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	"github.com/kubernetes-sigs/federation-v2/test/e2e/framework"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("Federated types", func() {
	f := framework.NewFederationFramework("federated-types")

	fedTypeConfigs := federatedtypes.FederatedTypeConfigs()
	for kind, _ := range fedTypeConfigs {
		// Bind the type config inside the loop to ensure the ginkgo
		// closure gets a different value for every loop iteration.
		//
		// Reference: https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		fedTypeConfig := fedTypeConfigs[kind]

		Describe(fmt.Sprintf("%q resources", kind), func() {
			It("should be created, read, updated and deleted successfully", func() {
				// TODO (font): e2e tests for federated Namespace using a
				// test managed federation does not work until k8s
				// namespace controller is added.
				if framework.TestContext.TestManagedFederation {
					if fedTypeConfig.Kind == federatedtypes.NamespaceKind {
						framework.Skipf("%s not supported for test managed federation.",
							fedTypeConfig.Kind)
					}
				} else {
					// TODO (font): e2e tests for federated replicasets and
					// deployments using a test unmanaged federation do not
					// work until comparison issues due to defaulting are
					// resolved.
					if fedTypeConfig.Kind == federatedtypes.FederatedReplicaSetKind ||
						fedTypeConfig.Kind == federatedtypes.FederatedDeploymentKind {
						framework.Skipf("%s not supported for test unmanaged federation.", fedTypeConfig.Kind)
					}
				}

				// Initialize an in-memory controller if configuration requires
				f.SetUpControllerFixture(fedTypeConfig.Kind, fedTypeConfig.AdapterFactory)

				userAgent := fmt.Sprintf("crud-test-%s", fedTypeConfig.Kind)
				adapter := fedTypeConfig.AdapterFactory(f.FedClient(userAgent))
				namespaceAdapter, ok := adapter.(*federatedtypes.FederatedNamespaceAdapter)
				if ok {
					namespaceAdapter.SetKubeClient(f.KubeClient(userAgent))
				}
				clusterClients := f.ClusterClients(userAgent)
				crudTester := common.NewFederatedTypeCrudTester(framework.NewE2ELogger(), adapter, clusterClients, framework.PollInterval, framework.SingleCallTimeout)
				clusterNames := []string{}
				for name, _ := range clusterClients {
					clusterNames = append(clusterNames, name)
				}
				template, placement, override := federatedtypes.NewTestObjects(fedTypeConfig.Kind, f.TestNamespaceName(), clusterNames)

				crudTester.CheckLifecycle(template, placement, override)
			})
		})
	}
})
