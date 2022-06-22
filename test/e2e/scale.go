/*
Copyright 2019 The Kubernetes Authors.

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
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"k8s.io/klog/v2"

	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/kubefed/pkg/apis/core/typeconfig"
	fedv1b1 "sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	genericclient "sigs.k8s.io/kubefed/pkg/client/generic"
	"sigs.k8s.io/kubefed/pkg/controller/util"
	"sigs.k8s.io/kubefed/pkg/kubefedctl"
	kfenable "sigs.k8s.io/kubefed/pkg/kubefedctl/enable"
	kfutil "sigs.k8s.io/kubefed/pkg/kubefedctl/util"
	"sigs.k8s.io/kubefed/test/common"
	"sigs.k8s.io/kubefed/test/e2e/framework"

	. "github.com/onsi/ginkgo" //nolint:stylecheck
)

// WARNING This test modifies the runtime behavior of the sync
// controller. Running it concurrently with other tests that use the
// sync controller is likely to result in unexpected behavior.

// This test is intended to validate the scale of kubefed across as
// many clusters as local resources allow.  Rather than impose the
// overhead of an apiserver per cluster, the host cluster is joined to
// itself for each cluster that needs to be simulated.
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
var _ = Describe("Simulated Scale", func() {
	baseName := "scale-test"
	f := framework.NewKubeFedFramework(baseName)

	tl := framework.NewE2ELogger()

	typeConfigFixtures := common.TypeConfigFixturesOrDie(tl)

	It("should create a simulated federation", func() {
		if !framework.TestContext.LimitedScope {
			framework.Skipf("Simulated scale testing is not compatible with cluster-scoped federation.")
		}
		if !framework.TestContext.InMemoryControllers {
			framework.Skipf("Simulated scale testing requires in-process controllers.")
		}

		client := f.KubeClient(baseName)

		// Create the host cluster namespace
		generateName := "scale-host-"
		hostNamespace, err := framework.CreateNamespace(client, generateName)
		if err != nil {
			tl.Fatalf("Error creating namespace: %v", err)
		}
		defer framework.DeleteNamespace(client, hostNamespace)
		hostCluster := hostNamespace

		// Reconfigure the test context to ensure the fixture setup
		// will work correctly with the simulated federation.
		framework.TestContext.KubeFedSystemNamespace = hostNamespace

		// Join the cluster to itself with the name of the cluster
		// being used as the federation namespace in each simulated
		// cluster.
		nameToken := strings.TrimPrefix(hostCluster, generateName)
		hostConfig := f.KubeConfig()
		memberClusters := []string{}
		for i := 0; i < framework.TestContext.ScaleClusterCount; i++ {
			memberCluster := fmt.Sprintf("scale-member-%d-%s", i, nameToken)
			memberClusters = append(memberClusters, memberCluster)
			joiningNamespace := memberCluster

			_, err := kubefedctl.TestOnlyJoinClusterForNamespace(
				hostConfig, hostConfig, hostNamespace,
				joiningNamespace, hostCluster, memberCluster,
				"", apiextv1.NamespaceScoped, false, false)

			defer func() {
				framework.DeleteNamespace(client, joiningNamespace)
			}()
			if err != nil {
				tl.Fatalf("Error joining cluster %s: %v", memberCluster, err)
			}
		}

		// rejoin errorOnExisting=false
		for i := 0; i < framework.TestContext.ScaleClusterCount; i++ {
			memberCluster := fmt.Sprintf("scale-member-rejoin-%d-%s", i, nameToken)
			memberClusters = append(memberClusters, memberCluster)
			joiningNamespace := memberCluster
			secretName := memberCluster

			_, errJoin := kubefedctl.TestOnlyJoinClusterForNamespace(
				hostConfig, hostConfig, hostNamespace,
				joiningNamespace, hostCluster, memberCluster,
				secretName, apiextv1.NamespaceScoped, false, false)

			// rejoin cluster, and secret not change
			_, errReJoin := kubefedctl.TestOnlyJoinClusterForNamespace(
				hostConfig, hostConfig, hostNamespace,
				joiningNamespace, hostCluster, memberCluster,
				secretName, apiextv1.NamespaceScoped, false, false)

			// serviceaccount token recreate
			saName := kfutil.ClusterServiceAccountName(memberCluster, hostCluster)
			var deleteSecret sync.Once
			err = wait.PollImmediate(1*time.Second, 10*time.Second, func() (bool, error) {
				sa, err := client.CoreV1().ServiceAccounts(joiningNamespace).Get(
					context.Background(), saName, metav1.GetOptions{},
				)
				if err != nil {
					return false, nil
				}

				// delete secret, make token regenerate
				deleteSecret.Do(func() {
					for _, objReference := range sa.Secrets {
						saSecretName := objReference.Name
						secret, err := client.CoreV1().Secrets(joiningNamespace).Get(
							context.Background(), saSecretName, metav1.GetOptions{},
						)
						if err != nil {
							tl.Fatalf("Error get sa secret %s: %v", saSecretName, err)
						}
						if secret.Type == corev1.SecretTypeServiceAccountToken {
							if err := client.CoreV1().Secrets(joiningNamespace).Delete(context.TODO(), saSecretName, metav1.DeleteOptions{}); err != nil {
								tl.Fatalf("Error delete secret %s: %v", secretName, err)
							}
						}
					}
				})
				for _, objReference := range sa.Secrets {
					saSecretName := objReference.Name
					secret, err := client.CoreV1().Secrets(joiningNamespace).Get(
						context.Background(), saSecretName, metav1.GetOptions{},
					)
					if err != nil {
						return false, nil
					}
					if secret.Type == corev1.SecretTypeServiceAccountToken {
						klog.V(2).Infof("Using secret named: %s", secret.Name)
						return true, nil
					}
				}
				return false, nil
			})

			// rejoin cluster, and secret change
			_, errReJoinAfterChange := kubefedctl.TestOnlyJoinClusterForNamespace(
				hostConfig, hostConfig, hostNamespace,
				joiningNamespace, hostCluster, memberCluster,
				secretName, apiextv1.NamespaceScoped, false, false)

			defer func() {
				framework.DeleteNamespace(client, joiningNamespace)
			}()
			if errJoin != nil {
				tl.Fatalf("Error joining cluster %s: %v", memberCluster, err)
			}
			if errReJoin != nil {
				tl.Fatalf("Error joining cluster %s: %v", memberCluster, err)
			}

			if errReJoinAfterChange != nil {
				tl.Fatalf("Error joining cluster %s: %v", memberCluster, err)
			}
		}
		// rejoin errorOnExisting=true
		for i := 0; i < framework.TestContext.ScaleClusterCount; i++ {
			memberCluster := fmt.Sprintf("scale-member-rejoin-erroronexisting-%d-%s", i, nameToken)
			memberClusters = append(memberClusters, memberCluster)
			joiningNamespace := memberCluster
			secretName := memberCluster

			_, errJoin := kubefedctl.TestOnlyJoinClusterForNamespace(
				hostConfig, hostConfig, hostNamespace,
				joiningNamespace, hostCluster, memberCluster,
				secretName, apiextv1.NamespaceScoped, false, true)

			_, errReJoin := kubefedctl.TestOnlyJoinClusterForNamespace(
				hostConfig, hostConfig, hostNamespace,
				joiningNamespace, hostCluster, memberCluster,
				secretName, apiextv1.NamespaceScoped, false, true)

			if errJoin != nil {
				tl.Fatalf("Error joining cluster %s: %v", memberCluster, err)
			}

			if errReJoin == nil {
				tl.Fatalf("Should error rejoining cluster %s", memberCluster)
			}
			klog.InfoS("expected error when rejoin cluster with errorOnExisting=true", "err", errReJoin)
		}

		// Override naming methods to allow the sync controller to
		// work with a simulated scale environment.

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

		// Enable federation of namespaces to ensure that the sync
		// controller for a namespaced type will be able to start.
		enableTypeFederation(tl, hostConfig, hostNamespace, "namespaces")

		// Enable federation of a namespaced type.
		targetType := "configmaps"
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
		crudTester, targetObject, overrides := initCrudTestWithPropagation(f, tl, f.KubeFedSystemNamespace(), typeConfig, testObjectsFunc, false)
		crudTester.CheckLifecycle(targetObject, overrides, nil)

		// Delete clusters to minimize errors logged by the cluster
		// controller.
		hostClient, err := genericclient.New(hostConfig)
		if err != nil {
			tl.Fatalf("Failed to get kubefed clientset: %v", err)
		}
		fedCluster := &fedv1b1.KubeFedCluster{}
		for _, memberCluster := range memberClusters {
			err := hostClient.Delete(context.TODO(), fedCluster, hostNamespace, memberCluster)
			if err != nil {
				tl.Errorf("Failed to delete cluster: %v", err)
			}
		}
	})
})

func enableTypeFederation(tl common.TestLogger, hostConfig *rest.Config, hostNamespace, targetType string) typeconfig.Interface {
	enableTypeDirective := kfenable.NewEnableTypeDirective()
	enableTypeDirective.Name = targetType
	resources, err := kfenable.GetResources(hostConfig, enableTypeDirective)
	if err != nil {
		tl.Fatalf("Error retrieving resources to enable federation of %q: %v", targetType, err)
	}
	err = kfenable.CreateResources(nil, hostConfig, resources, hostNamespace, false)
	if err != nil {
		tl.Fatalf("Error creating resources to enable federation of target type %q: %v", targetType, err)
	}
	// The created FederatedTypeConfig will be removed along with
	// the host namespace.  The CRD is not removed in case another
	// control plane is using it.
	return resources.TypeConfig
}
