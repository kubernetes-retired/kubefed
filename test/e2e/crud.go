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
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	"github.com/kubernetes-sigs/federation-v2/test/e2e/framework"

	. "github.com/onsi/ginkgo"
)

type testObjectAccessor func(namespace string, clusterNames []string) (fedObject *unstructured.Unstructured, err error)

var _ = Describe("Federated", func() {
	f := framework.NewFederationFramework("federated-types")

	tl := framework.NewE2ELogger()

	typeConfigFixtures := common.TypeConfigFixturesOrDie(tl)

	// TODO(marun) Ensure that deletion handling of federated
	// resources is performed in the event of test failure before
	// controllers are shutdown.

	for key := range typeConfigFixtures {
		typeConfigName := key
		fixture := typeConfigFixtures[key]
		Describe(fmt.Sprintf("%q", typeConfigName), func() {
			It("should be created, read, updated and deleted successfully", func() {
				// TODO (font): e2e tests for federated Namespace using a
				// test managed federation does not work until k8s
				// namespace controller is added.
				if typeConfigName == util.NamespaceName && framework.TestContext.TestManagedFederation {
					framework.Skipf("Federated %s not supported for test managed federation.", typeConfigName)
				}

				// Lookup the type config from the api
				dynClient, err := client.New(f.KubeConfig(), client.Options{})
				if err != nil {
					tl.Fatalf("Error initializing dynamic client: %v", err)
				}
				typeConfig := &fedv1a1.FederatedTypeConfig{}
				key := client.ObjectKey{Name: typeConfigName, Namespace: f.FederationSystemNamespace()}
				err = dynClient.Get(context.Background(), key, typeConfig)
				if err != nil {
					tl.Fatalf("Error retrieving federatedtypeconfig %q: %v", typeConfigName, err)
				}

				if framework.TestContext.LimitedScope && !typeConfig.GetNamespaced() {
					framework.Skipf("Federation of cluster-scoped type %s is not supported by a namespaced control plane.", typeConfigName)
				}

				testObjectFunc := func(namespace string, clusterNames []string) (*unstructured.Unstructured, error) {
					return common.NewTestObject(typeConfig, namespace, clusterNames, fixture)
				}
				orphanDependents := false
				validateCrud(f, tl, typeConfig, testObjectFunc, &orphanDependents)
				orphanDependents = true
				validateCrud(f, tl, typeConfig, testObjectFunc, &orphanDependents)
			})
		})
	}
})

func initCrudTest(f framework.FederationFramework, tl common.TestLogger,
	typeConfig typeconfig.Interface, testObjectFunc testObjectAccessor) (
	*common.FederatedTypeCrudTester, *unstructured.Unstructured) {

	// Initialize in-memory controllers if configuration requires
	fixture := f.SetUpSyncControllerFixture(typeConfig)
	f.RegisterFixture(fixture)

	if typeConfig.GetNamespaced() {
		// Propagation of namespaced types to member clusters depends on
		// their containing namespace being propagated.
		f.EnsureTestNamespacePropagation()
	}

	federatedKind := typeConfig.GetFederatedType().Kind

	userAgent := fmt.Sprintf("test-%s-crud", strings.ToLower(federatedKind))

	kubeConfig := f.KubeConfig()
	targetAPIResource := typeConfig.GetTarget()
	testClusters := f.ClusterDynamicClients(&targetAPIResource, userAgent)
	crudTester, err := common.NewFederatedTypeCrudTester(tl, typeConfig, kubeConfig, testClusters, framework.PollInterval, framework.TestContext.SingleCallTimeout)
	if err != nil {
		tl.Fatalf("Error creating crudtester for %q: %v", federatedKind, err)
	}

	namespace := ""
	// A test namespace is only required for namespaced resources or
	// namespaces themselves.
	if typeConfig.GetNamespaced() || typeConfig.GetTarget().Name == util.NamespaceName {
		namespace = f.TestNamespaceName()
	}

	clusterNames := []string{}
	for name := range testClusters {
		clusterNames = append(clusterNames, name)
	}
	fedObject, err := testObjectFunc(namespace, clusterNames)
	if err != nil {
		tl.Fatalf("Error creating test object: %v", err)
	}

	return crudTester, fedObject
}

func validateCrud(f framework.FederationFramework, tl common.TestLogger,
	typeConfig typeconfig.Interface, testObjectFunc testObjectAccessor,
	orphanDependents *bool) {

	crudTester, fedObject := initCrudTest(f, tl, typeConfig, testObjectFunc)
	crudTester.CheckLifecycle(fedObject, orphanDependents)
}
