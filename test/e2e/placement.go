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

package e2e

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	"github.com/kubernetes-sigs/federation-v2/test/e2e/framework"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("Placement", func() {
	f := framework.NewFederationFramework("placement")

	tl := framework.NewE2ELogger()

	typeConfigs := common.TypeConfigsOrDie(tl)

	// TODO(marun) Since this test only targets namespaced federation,
	// concurrent test isolation against unmanaged fixture is
	// effectively impossible.  The namespace placement would be
	// picked up by other controllers targeting the federation
	// namespace.
	It("should be computed from namespace and resource placement for namespaced federation", func() {
		if !framework.TestContext.LimitedScope {
			framework.Skipf("Considering namespace placement when determining resource placement is not supported for cluster-scoped federation.")
		}

		// Select the first non-namespace type config
		var selectedTypeConfig typeconfig.Interface
		for _, typeConfig := range typeConfigs {
			if typeConfig.GetTemplate().Kind != util.NamespaceKind {
				selectedTypeConfig = typeConfig
				break
			}
		}
		if selectedTypeConfig == nil {
			tl.Fatal("Unable to find non-namespace type config")
		}

		// Propagate a resource to member clusters
		testObjectFunc := func(namespace string, clusterNames []string) (template, placement, override *unstructured.Unstructured, err error) {
			return common.NewTestObjects(selectedTypeConfig, namespace, clusterNames)
		}
		crudTester, desiredTemplate, desiredPlacement, desiredOverride := initCrudTest(f, tl, selectedTypeConfig, testObjectFunc)
		template, _, _ := crudTester.CheckCreate(desiredTemplate, desiredPlacement, desiredOverride)
		defer func() {
			orphanDependents := false
			crudTester.CheckDelete(template, &orphanDependents)
		}()

		namespace := f.TestNamespaceName()

		// Create namespace placement with no clusters.
		client := f.FedClient("placement")
		placementKey := fmt.Sprintf("%s/%s", namespace, namespace)
		_, err := client.CoreV1alpha1().FederatedNamespacePlacements(namespace).Create(&fedv1a1.FederatedNamespacePlacement{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      namespace,
			},
		})
		if err != nil {
			tl.Fatalf("Error creating FederatedNamespacePlacement %s/%s: %v", placementKey, err)
		}
		tl.Logf("Created new FederatedNamespacePlacement %q", placementKey)
		// Ensure the removal of the namespace placement to avoid affecting other tests.
		defer func() {
			err := client.CoreV1alpha1().FederatedNamespacePlacements(namespace).Delete(namespace, nil)
			if err != nil && !errors.IsNotFound(err) {
				tl.Fatalf("Error deleting FederatedNamespacePlacement %s/%s: %v", namespace, namespace, err)
			}
			tl.Logf("Deleted FederatedNamespacePlacement %q", placementKey)
		}()

		// Check for removal of the propagated resource from all clusters
		targetAPIResource := selectedTypeConfig.GetTarget()
		targetKind := targetAPIResource.Kind
		qualifiedName := util.NewQualifiedName(template)
		for clusterName, testCluster := range crudTester.TestClusters() {
			client, err := util.NewResourceClientFromConfig(testCluster.Config, &targetAPIResource)
			if err != nil {
				tl.Fatalf("Error creating resource client for %q: %v", targetKind, err)
			}
			err = wait.PollImmediate(framework.PollInterval, framework.TestContext.SingleCallTimeout, func() (bool, error) {
				_, err := client.Resources(namespace).Get(qualifiedName.Name, metav1.GetOptions{})
				if errors.IsNotFound(err) {
					return true, nil
				}
				if err != nil {
					tl.Errorf("Failed to check for existence of %s %q in cluster %q: %v",
						targetKind, qualifiedName, clusterName, err,
					)
				}
				return false, nil
			})
			if err != nil {
				tl.Fatalf("Failed to confirm removal of %s %q in cluster %q within %v",
					targetKind, qualifiedName, clusterName, framework.TestContext.SingleCallTimeout,
				)
			}
			tl.Logf("Confirmed removal of %s %q in cluster %q",
				targetKind, qualifiedName, clusterName,
			)
		}
	})
})
