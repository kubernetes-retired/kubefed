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
	"time"

	apiv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeclientset "k8s.io/client-go/kubernetes"

	"sigs.k8s.io/kubefed/pkg/apis/core/typeconfig"
	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	genericclient "sigs.k8s.io/kubefed/pkg/client/generic"
	"sigs.k8s.io/kubefed/pkg/controller/sync/status"
	"sigs.k8s.io/kubefed/pkg/controller/util"
	"sigs.k8s.io/kubefed/pkg/kubefedctl/federate"
	"sigs.k8s.io/kubefed/test/common"
	"sigs.k8s.io/kubefed/test/e2e/framework"

	. "github.com/onsi/ginkgo" //nolint:stylecheck
)

var containedTypeNames = []string{"jobs.batch", "deployments.apps", "replicasets.apps"}

type testObjectsAccessor func(namespace string, clusterNames []string) (targetObject *unstructured.Unstructured, overrides []interface{}, err error)

var _ = Describe("Federated", func() {
	f := framework.NewKubeFedFramework("federated-types")

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
				typeConfig, testObjectsFunc := getCrudTestInput(f, tl, typeConfigName, fixture)
				crudTester, targetObject, overrides := initCrudTest(f, tl, f.KubeFedSystemNamespace(), typeConfig, testObjectsFunc)
				crudTester.CheckLifecycle(targetObject, overrides, nil)
			})

			for _, remoteStatusTypeName := range containedTypeNames {
				if typeConfigName == remoteStatusTypeName {

					It("should be created, read its remote status and deleted successfully", func() {
						kubeFedConfig := &v1beta1.KubeFedConfig{}
						client := genericclient.NewForConfigOrDie(f.KubeConfig())
						err := client.Get(context.TODO(), kubeFedConfig, f.KubeFedSystemNamespace(), util.KubeFedConfigName)
						if err != nil {
							tl.Fatalf("Error collecting the kubefedconfig file: %v", err)
						}
						tl.Logf("Show the content of the kubefedconfig file: '%v'", kubeFedConfig)

						typeConfig, testObjectsFunc := getCrudTestInput(f, tl, typeConfigName, fixture)
						crudTester, targetObject, overrides := initCrudTest(f, tl, f.KubeFedSystemNamespace(), typeConfig, testObjectsFunc)
						fedObject := crudTester.CheckCreate(targetObject, overrides, nil)

						By("Checking the remote status filled for each federated resource for every cluster")
						tl.Logf("Checking the existence of a remote status for each fedObj in every cluster: %v", fedObject)
						crudTester.CheckRemoteStatus(fedObject, targetObject)

						defer func() {
							crudTester.CheckDelete(fedObject, false)
						}()
					})
				}
			}

			// The tests that follow only need to be executed against
			// a single namespaced type.
			if typeConfigName != "configmaps" {
				return
			}

			It("should report NamespaceNotFederated in propagation status if the containing namespace is not federated", func() {
				if framework.TestContext.NamespaceScopedControlPlane() {
					framework.Skipf("Unable to test for NamespaceNotFederated for a namespace-scoped control plane")
				}

				typeConfig, testObjectsFunc := getCrudTestInput(f, tl, typeConfigName, fixture)
				// Initialize the test without creating a federated namespace.
				crudTester, targetObject, overrides := initCrudTestWithPropagation(f, tl, f.KubeFedSystemNamespace(), typeConfig, testObjectsFunc, false)

				kind := typeConfig.GetFederatedType().Kind

				By(fmt.Sprintf("Creating a %s whose containing namespace is not federated", kind))
				fedObject := crudTester.Create(targetObject, overrides, nil)

				qualifiedName := util.NewQualifiedName(fedObject)

				By(fmt.Sprintf("Waiting until the status of the %s %q indicates NamespaceNotFederated", kind, qualifiedName))
				client := genericclient.NewForConfigOrDie(f.KubeConfig())
				err := wait.PollImmediate(framework.PollInterval, wait.ForeverTestTimeout, func() (bool, error) {
					genericResource, err := common.GetGenericResource(client, fedObject.GroupVersionKind(), qualifiedName)
					if err != nil {
						tl.Fatalf("An error occurred retrieving the status of the %s %q: %v", kind, qualifiedName, err)
					}
					if genericResource.Status == nil {
						return false, nil
					}
					var propCondition *status.GenericCondition
					for _, condition := range genericResource.Status.Conditions {
						if condition.Type == status.PropagationConditionType {
							propCondition = condition
							break
						}
					}
					if propCondition == nil {
						return false, nil
					}
					return propCondition.Status == apiv1.ConditionFalse && propCondition.Reason == status.NamespaceNotFederated, nil
				})
				if err != nil {
					tl.Fatalf("Error waiting for %s %q to have propagation status NamespaceNotFederated: %v", kind, qualifiedName, err)
				}
			})

			It("should have the managed label removed if not managed", func() {
				typeConfig, testObjectsFunc := getCrudTestInput(f, tl, typeConfigName, fixture)
				crudTester, targetObject, _ := initCrudTest(f, tl, f.KubeFedSystemNamespace(), typeConfig, testObjectsFunc)

				testClusters := crudTester.TestClusters()

				By("Selecting a member cluster to create an unlabeled resource in")
				clusterName := ""
				for key := range testClusters {
					clusterName = key
					break
				}
				clusterConfig := testClusters[clusterName].Config

				By("Waiting for the test namespace to be created in the selected cluster")
				kubeClient := kubeclientset.NewForConfigOrDie(clusterConfig)
				common.WaitForNamespaceOrDie(tl, kubeClient, clusterName, targetObject.GetNamespace(),
					framework.PollInterval, framework.TestContext.SingleCallTimeout)

				By("Creating a labeled resource in the selected cluster")
				util.AddManagedLabel(targetObject)
				labeledObj, err := common.CreateResource(clusterConfig, typeConfig.GetTargetType(), targetObject)
				if err != nil {
					tl.Fatalf("Failed to create labeled resource in cluster %q: %v", clusterName, err)
				}
				clusterClient := genericclient.NewForConfigOrDie(clusterConfig)
				defer func() {
					err := clusterClient.Delete(context.TODO(), labeledObj, labeledObj.GetNamespace(), labeledObj.GetName())
					if err != nil {
						tl.Fatalf("Unexpected error: %v", err)
					}
				}()

				By("Checking that the labeled resource is unlabeled by the sync controller")
				err = wait.PollImmediate(framework.PollInterval, wait.ForeverTestTimeout, func() (bool, error) {
					obj := &unstructured.Unstructured{}
					obj.SetGroupVersionKind(labeledObj.GroupVersionKind())
					err := clusterClient.Get(context.TODO(), obj, labeledObj.GetNamespace(), labeledObj.GetName())
					if err != nil {
						tl.Errorf("Error retrieving labeled resource: %v", err)
						return false, nil
					}
					return !util.HasManagedLabel(obj), nil
				})
				if err != nil {
					tl.Fatal("Timed out waiting for the managed label to be removed")
				}
			})

			It("should not be deleted if unlabeled", func() {
				typeConfig, testObjectsFunc := getCrudTestInput(f, tl, typeConfigName, fixture)
				crudTester, targetObject, _ := initCrudTest(f, tl, f.KubeFedSystemNamespace(), typeConfig, testObjectsFunc)

				testClusters := crudTester.TestClusters()

				By("Selecting a member cluster to create an unlabeled resource in")
				clusterName := ""
				for key := range testClusters {
					clusterName = key
					break
				}
				clusterConfig := testClusters[clusterName].Config

				By("Waiting for the test namespace to be created in the selected cluster")
				kubeClient := kubeclientset.NewForConfigOrDie(clusterConfig)
				common.WaitForNamespaceOrDie(tl, kubeClient, clusterName, targetObject.GetNamespace(),
					framework.PollInterval, framework.TestContext.SingleCallTimeout)

				By("Creating an unlabeled resource in the selected cluster")
				unlabeledObj, err := common.CreateResource(clusterConfig, typeConfig.GetTargetType(), targetObject)
				if err != nil {
					tl.Fatalf("Failed to create unlabeled resource in cluster %q: %v", clusterName, err)
				}
				clusterClient := genericclient.NewForConfigOrDie(clusterConfig)
				defer func() {
					err := clusterClient.Delete(context.TODO(), unlabeledObj, unlabeledObj.GetNamespace(), unlabeledObj.GetName())
					if err != nil {
						tl.Fatalf("Unexpected error: %v", err)
					}
				}()

				By("Intitializing a federated resource with placement excluding all clusters")
				fedObject, err := federate.FederatedResourceFromTargetResource(typeConfig, unlabeledObj)
				if err != nil {
					tl.Fatalf("Error generating federated resource: %v", err)
				}
				err = util.SetClusterNames(fedObject, []string{})
				if err != nil {
					tl.Fatalf("Error setting cluster names for federated resource: %v", err)
				}
				fedObject.SetGenerateName("")

				By("Creating the federated resource")
				createdObj, err := common.CreateResource(f.KubeConfig(), typeConfig.GetFederatedType(), fedObject)
				if err != nil {
					tl.Fatalf("Error creating federated resource: %v", err)
				}
				hostClient := genericclient.NewForConfigOrDie(f.KubeConfig())
				defer func() {
					err := hostClient.Delete(context.TODO(), createdObj, createdObj.GetNamespace(), createdObj.GetName())
					if err != nil {
						tl.Fatalf("Unexpected error: %v", err)
					}
				}()

				waitDuration := 10 * time.Second // Arbitrary amount of time to wait for deletion
				By(fmt.Sprintf("Checking that the unlabeled resource is not deleted within %v", waitDuration))
				_ = wait.PollImmediate(framework.PollInterval, waitDuration, func() (bool, error) {
					obj := &unstructured.Unstructured{}
					obj.SetGroupVersionKind(unlabeledObj.GroupVersionKind())
					err := clusterClient.Get(context.TODO(), obj, unlabeledObj.GetNamespace(), unlabeledObj.GetName())
					if apierrors.IsNotFound(err) {
						tl.Fatalf("Unlabeled resource %s %q was deleted", typeConfig.GetTargetType().Kind, util.NewQualifiedName(unlabeledObj))
					}
					if err != nil {
						tl.Errorf("Error retrieving unlabeled resource: %v", err)
					}
					return false, nil
				})
			})
		})
	}
})

func getCrudTestInput(f framework.KubeFedFramework, tl common.TestLogger,
	typeConfigName string, fixture *unstructured.Unstructured) (
	typeconfig.Interface, testObjectsAccessor) {
	// Lookup the type config from the api
	client, err := genericclient.New(f.KubeConfig())
	if err != nil {
		tl.Fatalf("Error initializing dynamic client: %v", err)
	}
	typeConfig, err := common.GetTypeConfig(client, typeConfigName, f.KubeFedSystemNamespace())
	if err != nil {
		tl.Fatalf("Error retrieving federatedtypeconfig %q: %v", typeConfigName, err)
	}

	// Enable the Status Collection for this type of resource
	tc := typeConfig.(*v1beta1.FederatedTypeConfig)
	for _, typeName := range containedTypeNames {
		tl.Logf("TypeConfig name: %s", tc.GetName())
		if tc.GetName() == typeName {
			tl.Logf("Enabling remote status collection for %v", typeConfig.GetFederatedType().Kind)
			err = common.EnableStatusCollection(client, tc)
			if err != nil {
				tl.Fatalf("Error enabling the federatedtypeconfig %q: %v", typeConfigName, err)
			}
		}
	}

	if framework.TestContext.LimitedScope && !typeConfig.GetNamespaced() {
		framework.Skipf("Federation of cluster-scoped type %s is not supported by a namespaced control plane.", typeConfigName)
	}

	testObjectsFunc := func(namespace string, clusterNames []string) (*unstructured.Unstructured, []interface{}, error) {
		targetObject, err := common.NewTestTargetObject(typeConfig, namespace, fixture)
		if err != nil {
			return nil, nil, err
		}
		if typeConfig.GetTargetType().Kind == util.NamespaceKind {
			// Namespace crud testing needs to have the same name as its namespace.
			targetObject.SetName(namespace)
			targetObject.SetNamespace(namespace)
		}

		overrides, err := common.OverridesFromFixture(clusterNames, fixture)
		if err != nil {
			return nil, nil, err
		}
		return targetObject, overrides, err
	}
	return typeConfig, testObjectsFunc
}

func initCrudTest(f framework.KubeFedFramework, tl common.TestLogger, clustersNamespace string,
	typeConfig typeconfig.Interface, testObjectsFunc testObjectsAccessor) (
	*common.FederatedTypeCrudTester, *unstructured.Unstructured, []interface{}) {
	return initCrudTestWithPropagation(f, tl, clustersNamespace, typeConfig, testObjectsFunc, true)
}

func initCrudTestWithPropagation(f framework.KubeFedFramework, tl common.TestLogger, clustersNamespace string,
	typeConfig typeconfig.Interface, testObjectsFunc testObjectsAccessor,
	ensureNamespacePropagation bool) (
	*common.FederatedTypeCrudTester, *unstructured.Unstructured, []interface{}) {
	// Initialize in-memory controllers if configuration requires
	fixture := f.SetUpSyncControllerFixture(typeConfig)
	f.RegisterFixture(fixture)

	if typeConfig.GetNamespaced() && ensureNamespacePropagation {
		// Propagation of namespaced types to member clusters depends on
		// their containing namespace being propagated.
		f.EnsureTestNamespacePropagation()
	}

	federatedKind := typeConfig.GetFederatedType().Kind

	userAgent := fmt.Sprintf("test-%s-crud", strings.ToLower(federatedKind))

	kubeConfig := f.KubeConfig()
	targetAPIResource := typeConfig.GetTargetType()

	testClusters := f.ClusterDynamicClients(&targetAPIResource, userAgent)
	crudTester, err := common.NewFederatedTypeCrudTester(tl, typeConfig, kubeConfig, testClusters, clustersNamespace, framework.PollInterval, framework.TestContext.SingleCallTimeout)
	if err != nil {
		tl.Fatalf("Error creating crudtester for %q: %v", federatedKind, err)
	}

	namespace := ""
	// A test namespace is only required for namespaced resources or
	// namespaces themselves.
	if typeConfig.GetNamespaced() || typeConfig.GetTargetType().Name == util.NamespaceName {
		namespace = f.TestNamespaceName()
	}

	clusterNames := []string{}
	for name := range testClusters {
		clusterNames = append(clusterNames, name)
	}
	targetObject, overrides, err := testObjectsFunc(namespace, clusterNames)
	if err != nil {
		tl.Fatalf("Error creating test objects: %v", err)
	}

	return crudTester, targetObject, overrides
}
