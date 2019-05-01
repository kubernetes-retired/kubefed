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
	"fmt"

	"github.com/pkg/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	fedschedulingv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/scheduling/v1alpha1"
	genericclient "github.com/kubernetes-sigs/federation-v2/pkg/client/generic"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/schedulingmanager"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/kubefedctl"
	kfenable "github.com/kubernetes-sigs/federation-v2/pkg/kubefedctl/enable"
	"github.com/kubernetes-sigs/federation-v2/pkg/schedulingtypes"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	"github.com/kubernetes-sigs/federation-v2/test/e2e/framework"
	restclient "k8s.io/client-go/rest"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("Scheduling", func() {
	f := framework.NewFederationFramework("scheduling")
	tl := framework.NewE2ELogger()

	userAgent := "rsp-test"

	schedulingTypes := make(map[string]schedulingtypes.SchedulerFactory)
	for typeConfigName, schedulingType := range schedulingtypes.SchedulingTypes() {
		if schedulingType.Kind != schedulingtypes.RSPKind {
			continue
		}
		schedulingTypes[typeConfigName] = schedulingType.SchedulerFactory
	}
	if len(schedulingTypes) == 0 {
		tl.Fatalf("No target types found for scheduling type %q", schedulingtypes.RSPKind)
	}

	var kubeConfig *restclient.Config
	var genericClient genericclient.Client
	var namespace string
	var clusterNames []string
	var controllerFixture *framework.ControllerFixture
	var controller *schedulingmanager.SchedulingManager
	typeConfigs := make(map[string]typeconfig.Interface)

	BeforeEach(func() {
		// The following setup is shared across tests but must be
		// performed at test time rather than at test collection.
		if kubeConfig == nil {
			client, err := genericclient.New(f.KubeConfig())
			if err != nil {
				tl.Fatalf("Error initializing dynamic client: %v", err)
			}
			for targetTypeName := range schedulingTypes {
				typeConfig, err := common.GetTypeConfig(client, targetTypeName, f.FederationSystemNamespace())
				if err != nil {
					tl.Fatalf("Error retrieving federatedtypeconfig for %q: %v", targetTypeName, err)
				}
				typeConfigs[targetTypeName] = typeConfig
			}

			clusterNames = f.ClusterNames(userAgent)
			genericClient = f.Client(userAgent)
			kubeConfig = f.KubeConfig()
		}
		namespace = f.TestNamespaceName()
		if framework.TestContext.RunControllers() {
			controllerFixture, controller = framework.NewSchedulingManagerFixture(tl, f.ControllerConfig())
			f.RegisterFixture(controllerFixture)
		}
	})

	Describe("SchedulingManager", func() {
		Context("when federatedtypeconfig resources are changed", func() {
			It("related scheduler and plugin controllers should be dynamically disabled/enabled", func() {
				if !framework.TestContext.RunControllers() {
					framework.Skipf("The scheduling manager can only be tested when controllers are running in-process.")
				}

				// The deletion of FederatedTypeConfigs performed by this test
				// requires the FederatedTypeConfig controller in order to
				// remove its finalizer for proper deletion.
				controllerFixture = framework.NewFederatedTypeConfigControllerFixture(tl, f.ControllerConfig())
				f.RegisterFixture(controllerFixture)

				// make sure scheduler/plugin initialization are done before our test
				By("Waiting for scheduler/plugin controllers are initialized in scheduling manager")
				waitForSchedulerStarted(tl, controller, schedulingTypes)

				By("Deleting federatedtypeconfig resources for scheduler/plugin controllers")
				for targetTypeName := range schedulingTypes {
					deleteTypeConfigResource(targetTypeName, f.FederationSystemNamespace(), kubeConfig, tl)
				}

				By("Waiting for scheduler/plugin controllers are destroyed in scheduling manager")
				waitForSchedulerDeleted(tl, controller, schedulingTypes)

				By("Enabling federatedtypeconfig resources again for scheduler/plugin controllers")
				for targetTypeName := range schedulingTypes {
					enableTypeConfigResource(targetTypeName, f.FederationSystemNamespace(), kubeConfig, tl)
				}

				By("Waiting for the scheduler/plugin controllers are started in scheduling manager")
				waitForSchedulerStarted(tl, controller, schedulingTypes)
			})
		})
	})

	Describe("ReplicaSchedulingPreferences", func() {
		testCases := map[string]struct {
			total         int32
			weight1       int64
			weight2       int64
			min1          int64
			min2          int64
			cluster1      int32
			cluster2      int32
			noPreferences bool
		}{
			"replicas spread equally in clusters with no explicit per cluster preferences": {
				total:         int32(4),
				cluster1:      int32(2),
				cluster2:      int32(2),
				noPreferences: true,
			},
			"replicas spread in proportion of weights when explicit preferences with weights specified": {
				total:    int32(6),
				weight1:  int64(2),
				weight2:  int64(1),
				min1:     int64(0),
				min2:     int64(0),
				cluster1: int32(4),
				cluster2: int32(2),
			},
			"replicas spread considering min replicas when both minreplica and weights specified": {
				total:    int32(6),
				weight1:  int64(2),
				weight2:  int64(1),
				min1:     int64(3),
				min2:     int64(3),
				cluster1: int32(3),
				cluster2: int32(3),
			},
		}

		for key := range schedulingTypes {
			typeConfigName := key

			Describe(fmt.Sprintf("scheduling for federated %s", typeConfigName), func() {
				for testName, tc := range testCases {
					It(fmt.Sprintf("should result in %s", testName), func() {

						typeConfig, ok := typeConfigs[typeConfigName]
						if !ok {
							tl.Fatalf("Unable to find type config for %q", typeConfigName)
						}
						federatedKind := typeConfig.GetFederatedType().Kind

						clusterCount := len(clusterNames)
						if clusterCount != 2 {
							framework.Skipf("Tests of ReplicaSchedulingPreferences requires 2 clusters but got: %d", clusterCount)
						}

						var rspSpec fedschedulingv1a1.ReplicaSchedulingPreferenceSpec
						if tc.noPreferences {
							rspSpec = rspSpecWithoutClusterList(tc.total, federatedKind)
						} else {
							rspSpec = rspSpecWithClusterList(tc.total, tc.weight1, tc.weight2, tc.min1, tc.min2, clusterNames, federatedKind)
						}

						expected := map[string]int32{
							clusterNames[0]: tc.cluster1,
							clusterNames[1]: tc.cluster2,
						}

						name, err := createTestObjs(tl, genericClient, typeConfig, kubeConfig, rspSpec, namespace)
						if err != nil {
							tl.Fatalf("Creation of test objects failed in federation: %v", err)
						}

						err = waitForMatchingFederatedObject(tl, typeConfig, kubeConfig, name, namespace, expected)
						if err != nil {
							tl.Fatalf("Failed waiting for matching federated object: %v", err)
						}

						err = deleteTestObj(typeConfig, kubeConfig, name, namespace)
						if err != nil {
							tl.Fatalf("Deletion of test object failed in fedeartion: %v", err)
						}
					})
				}
			})
		}
	})
})

func waitForSchedulerDeleted(tl common.TestLogger, controller *schedulingmanager.SchedulingManager, schedulingTypes map[string]schedulingtypes.SchedulerFactory) {
	err := wait.PollImmediate(framework.PollInterval, framework.TestContext.SingleCallTimeout, func() (bool, error) {
		scheduler := controller.GetScheduler(schedulingtypes.RSPKind)
		if scheduler != nil {
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		tl.Fatalf("Error stopping for scheduler/plugin controllers: %v", err)
	}
}

func waitForSchedulerStarted(tl common.TestLogger, controller *schedulingmanager.SchedulingManager, schedulingTypes map[string]schedulingtypes.SchedulerFactory) {
	err := wait.PollImmediate(framework.PollInterval, framework.TestContext.SingleCallTimeout, func() (bool, error) {
		scheduler := controller.GetScheduler(schedulingtypes.RSPKind)
		if scheduler == nil {
			return false, nil
		}
		for targetTypeName := range schedulingTypes {
			if !scheduler.HasPlugin(targetTypeName) {
				return false, nil
			}
		}

		return true, nil
	})
	if err != nil {
		tl.Fatalf("Error starting for scheduler and plugins: %v", err)
	}
}

func rspSpecWithoutClusterList(total int32, targetKind string) fedschedulingv1a1.ReplicaSchedulingPreferenceSpec {
	return fedschedulingv1a1.ReplicaSchedulingPreferenceSpec{
		TotalReplicas: total,
		TargetKind:    targetKind,
		Clusters:      map[string]fedschedulingv1a1.ClusterPreferences{},
	}
}

// This assumes test setup using 2 clusters
func rspSpecWithClusterList(total int32, w1, w2, min1, min2 int64, clusters []string, targetKind string) fedschedulingv1a1.ReplicaSchedulingPreferenceSpec {
	rspSpec := rspSpecWithoutClusterList(total, targetKind)
	rspSpec.Clusters = map[string]fedschedulingv1a1.ClusterPreferences{
		clusters[0]: {
			MinReplicas: min1,
			Weight:      w1,
		},
		clusters[1]: {
			MinReplicas: min2,
			Weight:      w2,
		},
	}

	return rspSpec
}

func createTestObjs(tl common.TestLogger, client genericclient.Client, typeConfig typeconfig.Interface, kubeConfig *restclient.Config, rspSpec fedschedulingv1a1.ReplicaSchedulingPreferenceSpec, namespace string) (string, error) {
	federatedTypeAPIResource := typeConfig.GetFederatedType()
	federatedTypeClient, err := util.NewResourceClient(kubeConfig, &federatedTypeAPIResource)
	if err != nil {
		return "", err
	}
	typeConfigFixtures := common.TypeConfigFixturesOrDie(tl)
	typeConfigName := typeConfig.GetObjectMeta().Name
	fixture, ok := typeConfigFixtures[typeConfigName]
	if !ok {
		return "", errors.Errorf("Unable to find fixture for %q", typeConfigName)
	}
	fedObject, err := common.NewTestObject(typeConfig, namespace, []string{}, fixture)
	if err != nil {
		return "", err
	}
	createdFedObject, err := federatedTypeClient.Resources(namespace).Create(fedObject, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}
	name := createdFedObject.GetName()

	rsp := &fedschedulingv1a1.ReplicaSchedulingPreference{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: rspSpec,
	}
	err = client.Create(context.TODO(), rsp)
	if err != nil {
		return "", err
	}

	return name, nil
}

func deleteTestObj(typeConfig typeconfig.Interface, kubeConfig *restclient.Config, name, namespace string) error {
	federatedTypeAPIResource := typeConfig.GetFederatedType()
	federatedTypeClient, err := util.NewResourceClient(kubeConfig, &federatedTypeAPIResource)
	if err != nil {
		return err
	}

	err = federatedTypeClient.Resources(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}

func waitForMatchingFederatedObject(tl common.TestLogger, typeConfig typeconfig.Interface, kubeConfig *restclient.Config, name, namespace string, expected32 map[string]int32) error {
	apiResource := typeConfig.GetFederatedType()
	kind := apiResource.Kind
	client, err := util.NewResourceClient(kubeConfig, &apiResource)
	if err != nil {
		return err
	}

	expectedClusterNames := []string{}
	for clusterName := range expected32 {
		expectedClusterNames = append(expectedClusterNames, clusterName)
	}

	expected64 := int32MapToInt64(expected32)

	return wait.PollImmediate(framework.PollInterval, framework.TestContext.SingleCallTimeout, func() (bool, error) {
		fedObject, err := client.Resources(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				tl.Errorf("An error occurred while polling for %s %s/%s: %v", kind, namespace, name, err)
			}
			return false, nil
		}

		clusterNames, err := util.GetClusterNames(fedObject)
		if err != nil {
			tl.Errorf("An error occurred while retrieving cluster names for override %s %s/%s: %v", kind, namespace, name, err)
			return false, nil
		}
		if schedulingtypes.PlacementUpdateNeeded(clusterNames, expectedClusterNames) {
			return false, nil
		}

		overridesMap, err := util.GetOverrides(fedObject)
		if err != nil {
			tl.Errorf("Error reading cluster overrides for %s %s/%s: %v", kind, namespace, name, err)
			return false, nil
		}
		return !schedulingtypes.OverrideUpdateNeeded(overridesMap, expected64), nil
	})
}

func int32MapToInt64(original map[string]int32) map[string]int64 {
	result := make(map[string]int64)
	for k, v := range original {
		result[k] = int64(v)
	}
	return result
}

func enableTypeConfigResource(name, namespace string, config *restclient.Config, tl common.TestLogger) {
	for _, enableTypeDirective := range framework.LoadEnableTypeDirectives(tl) {
		resources, err := kfenable.GetResources(config, enableTypeDirective)
		if err != nil {
			tl.Fatalf("Error retrieving resource definitions for EnableTypeDirective %q: %v", enableTypeDirective.Name, err)
		}

		if enableTypeDirective.Name == name {
			err = kfenable.CreateResources(nil, config, resources, namespace)
			if err != nil {
				tl.Fatalf("Error creating resources for EnableTypeDirective %q: %v", enableTypeDirective.Name, err)
			}
		}
	}
}

func deleteTypeConfigResource(name, namespace string, config *restclient.Config, tl common.TestLogger) {
	qualifiedName := util.QualifiedName{Namespace: namespace, Name: name}
	err := kubefedctl.DisableFederation(nil, config, nil, qualifiedName, true, false, false)
	if err != nil {
		tl.Fatalf("Error disabling federation of target type %q: %v", qualifiedName, err)
	}
}
