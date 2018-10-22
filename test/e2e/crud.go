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
	"strings"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	"github.com/kubernetes-sigs/federation-v2/test/e2e/framework"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	. "github.com/onsi/ginkgo"
)

type testObjectAccessor func(namespace string, clusterNames []string) (template, placement, override *unstructured.Unstructured, err error)

var _ = Describe("Federated types", func() {
	f := framework.NewFederationFramework("federated-types")

	tl := framework.NewE2ELogger()

	typeConfigs := common.TypeConfigsOrDie(tl)

	for i, _ := range typeConfigs {
		// Bind the type config inside the loop to ensure the ginkgo
		// closure gets a different value for every loop iteration.
		//
		// Reference: https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		typeConfig := typeConfigs[i]
		templateKind := typeConfig.GetTemplate().Kind

		Describe(fmt.Sprintf("%q resources", templateKind), func() {
			It("should be created, read, updated and deleted successfully", func() {
				if templateKind == util.NamespaceKind {
					// TODO (font): e2e tests for federated Namespace using a
					// test managed federation does not work until k8s
					// namespace controller is added.
					if framework.TestContext.TestManagedFederation {
						framework.Skipf("%s not supported for test managed federation.", templateKind)
					}
					if framework.TestContext.LimitedScope {
						// It is not possible to propagate namespaces when namespaced.
						framework.Skipf("%s federation not supported for namespaced control plane.", templateKind)
					}
				}

				testObjectFunc := func(namespace string, clusterNames []string) (template, placement, override *unstructured.Unstructured, err error) {
					return common.NewTestObjects(typeConfig, namespace, clusterNames)
				}
				validateCrud(f, tl, typeConfig, testObjectFunc)
			})
		})
	}
})

func initCrudTest(f framework.FederationFramework, tl common.TestLogger,
	typeConfig typeconfig.Interface, testObjectFunc testObjectAccessor) (
	crudTester *common.FederatedTypeCrudTester, template, placement,
	override *unstructured.Unstructured) {

	// Initialize in-memory controllers if configuration requires
	f.SetUpControllerFixture(typeConfig)

	templateKind := typeConfig.GetTemplate().Kind

	userAgent := fmt.Sprintf("test-%s-crud", strings.ToLower(templateKind))

	kubeConfig := f.KubeConfig()
	targetAPIResource := typeConfig.GetTarget()
	testClusters := f.ClusterDynamicClients(&targetAPIResource, userAgent)
	crudTester, err := common.NewFederatedTypeCrudTester(tl, typeConfig, kubeConfig, testClusters, framework.PollInterval, framework.TestContext.SingleCallTimeout)
	if err != nil {
		tl.Fatalf("Error creating crudtester for %q: %v", templateKind, err)
	}

	clusterNames := []string{}
	for name, _ := range testClusters {
		clusterNames = append(clusterNames, name)
	}
	template, placement, override, err = testObjectFunc(f.TestNamespaceName(), clusterNames)
	if err != nil {
		tl.Fatalf("Error creating test objects: %v", err)
	}

	return crudTester, template, placement, override
}

func validateCrud(f framework.FederationFramework, tl common.TestLogger,
	typeConfig typeconfig.Interface, testObjectFunc testObjectAccessor) {

	crudTester, template, placement, override := initCrudTest(f, tl, typeConfig, testObjectFunc)
	crudTester.CheckLifecycle(template, placement, override)
}
