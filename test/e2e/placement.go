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
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"

	"sigs.k8s.io/kubefed/pkg/apis/core/typeconfig"
	fedv1b1 "sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	genericclient "sigs.k8s.io/kubefed/pkg/client/generic"
	"sigs.k8s.io/kubefed/pkg/controller/util"
	"sigs.k8s.io/kubefed/test/common"
	"sigs.k8s.io/kubefed/test/e2e/framework"

	. "github.com/onsi/ginkgo" //nolint:stylecheck
)

var _ = Describe("Placement", func() {
	f := framework.NewKubeFedFramework("placement")

	tl := framework.NewE2ELogger()

	typeConfigFixtures := common.TypeConfigFixturesOrDie(tl)

	// TODO(marun) Since this test only targets a namespaced control
	// plane, concurrent test isolation against unmanaged fixture is
	// effectively impossible.  The namespace placement would be
	// picked up by other controllers targeting the KubeFed system
	// namespace.
	It("should be computed from namespace and resource placement for a namespaced control plane", func() {
		if !framework.TestContext.LimitedScope {
			framework.Skipf("Considering namespace placement when determining resource placement is not supported for a cluster-scoped control plane.")
		}

		client, err := genericclient.New(f.KubeConfig())
		if err != nil {
			tl.Fatalf("Error initializing dynamic client: %v", err)
		}

		// Select the first namespaced type config
		var selectedTypeConfig typeconfig.Interface
		var fixture *unstructured.Unstructured
		for typeConfigName, typeConfigFixture := range typeConfigFixtures {
			typeConfig := &fedv1b1.FederatedTypeConfig{}
			err = client.Get(context.Background(), typeConfig, f.KubeFedSystemNamespace(), typeConfigName)
			if errors.IsNotFound(err) {
				continue
			}
			if err != nil {
				tl.Fatalf("Error retrieving federatedtypeconfig %q: %v", typeConfigName, err)
			}
			if !typeConfig.GetNamespaced() {
				continue
			}
			selectedTypeConfig = typeConfig
			fixture = typeConfigFixture
			break
		}
		if selectedTypeConfig == nil {
			tl.Fatal("Unable to find non-namespace type config")
		}

		// Propagate a resource to member clusters
		testObjectsFunc := func(namespace string, clusterNames []string) (*unstructured.Unstructured, []interface{}, error) {
			targetObject, err := common.NewTestTargetObject(selectedTypeConfig, namespace, fixture)
			if err != nil {
				return nil, nil, err
			}
			return targetObject, nil, err
		}
		crudTester, desiredTargetObject, _ := initCrudTest(f, tl, f.KubeFedSystemNamespace(), selectedTypeConfig, testObjectsFunc)
		fedObject := crudTester.CheckCreate(desiredTargetObject, nil, nil)
		defer func() {
			crudTester.CheckDelete(fedObject, false)
		}()

		// Wait until pending events for the templates have cleared
		// from the controller queue to ensure that event handling for
		// namespace placement is tested.  If a reconcile event
		// remains in the queue a resource may be reconciled even in
		// the absence of reconcile events being queued by a namespace
		// placement event.
		//
		// TODO(marun) This is non-deterministic, revisit if it ends up being flakey.
		time.Sleep(5 * time.Second)

		namespace := f.TestNamespaceName()

		// Ensure federated namespace with placement selecting no
		// clusters exist for the test namespace.
		fedNamespace := f.EnsureTestFederatedNamespace(false)
		fedNamespaceKey := util.NewQualifiedName(fedNamespace).String()
		// Ensure the removal of the namespace placement to avoid affecting other tests.
		defer func() {
			err := client.Delete(context.Background(), fedNamespace, fedNamespace.GetNamespace(), fedNamespace.GetName())
			if err != nil && !errors.IsNotFound(err) {
				tl.Fatalf("Error deleting FederatedNamespace %q: %v", fedNamespaceKey, err)
			}
			tl.Logf("Deleted FederatedNamespace %q", fedNamespaceKey)
		}()

		// Check for removal of the propagated resource from all clusters
		targetAPIResource := selectedTypeConfig.GetTargetType()
		targetKind := targetAPIResource.Kind
		qualifiedName := util.NewQualifiedName(fedObject)
		for clusterName, testCluster := range crudTester.TestClusters() {
			client, err := util.NewResourceClient(testCluster.Config, &targetAPIResource)
			if err != nil {
				tl.Fatalf("Error creating resource client for %q: %v", targetKind, err)
			}
			err = wait.PollImmediate(framework.PollInterval, framework.TestContext.SingleCallTimeout, func() (bool, error) {
				_, err := client.Resources(namespace).Get(context.Background(), qualifiedName.Name, metav1.GetOptions{})
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
