/*
Copyright 2018 The Federation v2 Authors.

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
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	fedschedulingv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/scheduling/v1alpha1"
	clientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/schedulingtypes"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	"github.com/kubernetes-sigs/federation-v2/test/e2e/framework"
	"github.com/kubernetes-sigs/federation-v2/test/e2e/framework/managed"
	restclient "k8s.io/client-go/rest"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("ReplicaSchedulingPreferences", func() {
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
	var fedClient fedclientset.Interface
	var namespace string
	var clusterNames []string
	typeConfigs := make(map[string]typeconfig.Interface)

	BeforeEach(func() {
		// The following setup is shared across tests but must be
		// performed at test time rather than at test collection.
		if kubeConfig == nil {
			dynClient, err := client.New(f.KubeConfig(), client.Options{})
			if err != nil {
				tl.Fatalf("Error initializing dynamic client: %v", err)
			}
			for targetTypeName := range schedulingTypes {
				typeConfig := &fedv1a1.FederatedTypeConfig{}
				key := client.ObjectKey{
					Namespace: f.FederationSystemNamespace(),
					Name:      targetTypeName,
				}
				err = dynClient.Get(context.Background(), key, typeConfig)
				if err != nil {
					tl.Fatalf("Error retrieving federatedtypeconfig for %q: %v", targetTypeName, err)
				}
				typeConfigs[targetTypeName] = typeConfig
			}

			clusterNames = f.ClusterNames(userAgent)
			fedClient = f.FedClient(userAgent)
			kubeConfig = f.KubeConfig()
		}
		namespace = f.TestNamespaceName()
		if framework.TestContext.RunControllers() {
			fixture := managed.NewSchedulerControllerFixture(tl, f.ControllerConfig())
			f.RegisterFixture(fixture)
		}
	})

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

					name, err := createTestObjs(tl, fedClient, typeConfig, kubeConfig, rspSpec, namespace)
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

func createTestObjs(tl common.TestLogger, fedClient clientset.Interface, typeConfig typeconfig.Interface, kubeConfig *restclient.Config, rspSpec fedschedulingv1a1.ReplicaSchedulingPreferenceSpec, namespace string) (string, error) {
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
	_, err = fedClient.SchedulingV1alpha1().ReplicaSchedulingPreferences(namespace).Create(rsp)
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
