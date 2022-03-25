/*
Copyright 2022 The Kubernetes Authors.

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

	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	fedv1b1 "sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	genericclient "sigs.k8s.io/kubefed/pkg/client/generic"
	"sigs.k8s.io/kubefed/pkg/controller/util"
	"sigs.k8s.io/kubefed/pkg/kubefedctl"
	"sigs.k8s.io/kubefed/test/common"
	"sigs.k8s.io/kubefed/test/e2e/framework"

	. "github.com/onsi/ginkgo" //nolint:stylecheck
)

// WARNING This test modifies the runtime behavior of the sync
// controller. Running it concurrently with other tests that use the
// sync controller is likely to result in unexpected behavior.

// This test is intended to validate CRUD operations even in the
// presence of not ready federated clusters if they are not the target
// of the operations. The test creates multiple self joins to simulate
// healthy and unhealthy clusters.
//
// Usually joining a cluster creates a namespace with the same name as
// the kubefed system namespace in the host cluster.  To support
// multiple self-joins, the kubefed namespace in member clusters needs
// to vary by join.
//
// This test needs to run namespaced controllers since each cluster
// will be simulated by a single namespace in the host cluster.  The
// name of each member cluster's namespace should match the name of
// the member cluster.
//
var _ = Describe("[NOT_READY] Simulated not-ready nodes", func() {
	baseName := "unhealthy-test"
	f := framework.NewKubeFedFramework(baseName)

	tl := framework.NewE2ELogger()

	typeConfigFixtures := common.TypeConfigFixturesOrDie(tl)

	It("should simulate unhealthy clusters", func() {
		if !framework.TestContext.LimitedScope {
			framework.Skipf("Simulated scale testing is not compatible with cluster-scoped federation.")
		}
		if !framework.TestContext.InMemoryControllers {
			framework.Skipf("Simulated scale testing requires in-process controllers.")
		}

		client := f.KubeClient(baseName)

		// Create the host cluster namespace
		generateName := "unhealthy-host-"
		hostNamespace, err := framework.CreateNamespace(client, generateName)
		if err != nil {
			tl.Fatalf("Error creating namespace: %v", err)
		}
		defer framework.DeleteNamespace(client, hostNamespace)

		// Reconfigure the test context to ensure the fixture setup
		// will work correctly with the simulated federation.
		framework.TestContext.KubeFedSystemNamespace = hostNamespace
		hostConfig := f.KubeConfig()

		unhealthyCluster := "unhealthy"
		_, err = kubefedctl.TestOnlyJoinClusterForNamespace(hostConfig, hostConfig, hostNamespace, unhealthyCluster, hostNamespace, unhealthyCluster, "", apiextv1.NamespaceScoped, false, false)
		if err != nil {
			tl.Fatalf("Error joining unhealthy cluster: %v", err)
		}

		healthyCluster := "healthy"
		_, err = kubefedctl.TestOnlyJoinClusterForNamespace(hostConfig, hostConfig, hostNamespace, healthyCluster, hostNamespace, healthyCluster, "", apiextv1.NamespaceScoped, false, false)
		if err != nil {
			tl.Fatalf("Error joining healthy cluster: %v", err)
		}
		hostClient, err := genericclient.New(hostConfig)
		if err != nil {
			tl.Fatalf("Failed to get kubefed clientset: %v", err)
		}
		healthyFedCluster := &unstructured.Unstructured{}
		healthyFedCluster.SetGroupVersionKind(schema.GroupVersionKind{
			Kind:    "KubeFedCluster",
			Group:   fedv1b1.SchemeGroupVersion.Group,
			Version: fedv1b1.SchemeGroupVersion.Version,
		})

		err = hostClient.Get(context.Background(), healthyFedCluster, hostNamespace, healthyCluster)
		if err != nil {
			tl.Fatalf("Cannot get healthyCluster: %v", err)
		}
		addLabel(healthyFedCluster, "healthy", "true")
		err = hostClient.Update(context.TODO(), healthyFedCluster)
		if err != nil {
			tl.Fatalf("Error updating label for healthy cluster: %v", err)
		}

		// Override naming methods to allow the sync controller to
		// work with a simulated federation environment.
		oldNamespaceForCluster := util.NamespaceForCluster
		util.NamespaceForCluster = func(clusterName, namespace string) string {
			return clusterName
		}
		defer func() {
			util.NamespaceForCluster = oldNamespaceForCluster
		}()

		oldNamespaceForResource := util.NamespaceForResource
		util.NamespaceForResource = func(resourceNamespace, fedNamespace string) string {
			return fedNamespace
		}
		defer func() {
			util.NamespaceForResource = oldNamespaceForResource
		}()
		oldQualifiedNameForCluster := util.QualifiedNameForCluster
		util.QualifiedNameForCluster = func(clusterName string, qualifiedName util.QualifiedName) util.QualifiedName {
			return util.QualifiedName{
				Name:      qualifiedName.Name,
				Namespace: clusterName,
			}
		}
		defer func() {
			util.QualifiedNameForCluster = oldQualifiedNameForCluster
		}()

		// Ensure that the cluster controller is able to successfully
		// health check the simulated clusters.
		framework.SetUpControlPlane()
		framework.WaitForUnmanagedClusterReadiness()

		fedCluster := &fedv1b1.KubeFedCluster{
			ObjectMeta: v1.ObjectMeta{
				Name:      unhealthyCluster,
				Namespace: hostNamespace,
			},
		}
		err = hostClient.Patch(context.Background(), fedCluster, runtimeclient.RawPatch(types.MergePatchType, []byte(`{"spec": {"apiEndpoint": "http://invalid_adress"}}`)))
		if err != nil {
			tl.Fatalf("Failed to patch kubefed cluster: %v", err)
		}

		err = wait.Poll(time.Second*5, time.Second*30, func() (bool, error) {
			cluster := &fedv1b1.KubeFedCluster{}
			err := hostClient.Get(context.TODO(), cluster, hostNamespace, unhealthyCluster)
			if err != nil {
				tl.Fatalf("Failed to retrieve unhealthy cluster: %v", err)
			}
			return !util.IsClusterReady(&cluster.Status), nil
		})
		if err != nil {
			tl.Fatalf("Error waiting for unhealthy cluster: %v", err)
		}

		// Enable federation of namespaces to ensure that the sync
		// controller for a namespaced type will be able to start.
		enableTypeFederation(tl, hostConfig, hostNamespace, "namespaces")

		// Enable federation of a namespaced type.
		targetType := "secrets"
		typeConfig := enableTypeFederation(tl, hostConfig, hostNamespace, targetType)
		// Perform crud testing for the type
		testObjectsFunc := func(namespace string, clusterNames []string) (*unstructured.Unstructured, []interface{}, error) {
			fixture := typeConfigFixtures[targetType]
			targetObject, err := common.NewTestTargetObject(typeConfig, namespace, fixture)
			if err != nil {
				return nil, nil, err
			}
			return targetObject, nil, err
		}
		crudTester, targetObject, overrides := initCrudTestWithPropagation(f, tl, hostNamespace, typeConfig, testObjectsFunc, false)
		fedObject := crudTester.CheckCreate(targetObject, overrides, map[string]string{"healthy": "true"})
		crudTester.CheckStatusCreated(util.NewQualifiedName(fedObject))
		crudTester.CheckUpdate(fedObject)
		crudTester.CheckDelete(fedObject, false)
	})
})
