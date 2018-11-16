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

	"k8s.io/apimachinery/pkg/api/errors"
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
					templateKind := typeConfig.GetTemplate().Kind

					clusterCount := len(clusterNames)
					if clusterCount != 2 {
						framework.Skipf("Tests of ReplicaSchedulingPreferences requires 2 clusters but got: %d", clusterCount)
					}

					var rspSpec fedschedulingv1a1.ReplicaSchedulingPreferenceSpec
					if tc.noPreferences {
						rspSpec = rspSpecWithoutClusterList(tc.total, templateKind)
					} else {
						rspSpec = rspSpecWithClusterList(tc.total, tc.weight1, tc.weight2, tc.min1, tc.min2, clusterNames, templateKind)
					}

					expected := map[string]int32{
						clusterNames[0]: tc.cluster1,
						clusterNames[1]: tc.cluster2,
					}

					name, err := createTestObjs(fedClient, typeConfig, kubeConfig, rspSpec, namespace)
					if err != nil {
						tl.Fatalf("Creation of test objects failed in federation: %v", err)
					}

					err = waitForMatchingPlacement(tl, typeConfig, kubeConfig, name, namespace, expected)
					if err != nil {
						tl.Fatalf("Failed waiting for matching placements: %v", err)
					}

					err = waitForMatchingOverride(tl, typeConfig, kubeConfig, name, namespace, expected)
					if err != nil {
						tl.Fatalf("Failed waiting for matching overrides: %v", err)
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

func createTestObjs(fedClient clientset.Interface, typeConfig typeconfig.Interface, kubeConfig *restclient.Config, rspSpec fedschedulingv1a1.ReplicaSchedulingPreferenceSpec, namespace string) (string, error) {
	templateAPIResource := typeConfig.GetTemplate()
	templateClient, err := util.NewResourceClientFromConfig(kubeConfig, &templateAPIResource)
	if err != nil {
		return "", err
	}
	// TODO(marun) retrieve fixture centrally
	typeConfigFixtures, err := common.TypeConfigFixtures()
	if err != nil {
		return "", fmt.Errorf("Error loading type config fixture: %v", err)
	}
	typeConfigName := typeConfig.GetObjectMeta().Name
	fixture, ok := typeConfigFixtures[typeConfigName]
	if !ok {
		return "", fmt.Errorf("Unable to find fixture for %q", typeConfigName)
	}
	template, err := common.NewTestTemplate(typeConfig.GetTemplate(), namespace, fixture)
	if err != nil {
		return "", err
	}
	createdTemplate, err := templateClient.Resources(namespace).Create(template)
	if err != nil {
		return "", err
	}
	name := createdTemplate.GetName()

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

func waitForMatchingPlacement(tl common.TestLogger, typeConfig typeconfig.Interface, kubeConfig *restclient.Config, name, namespace string, expected map[string]int32) error {
	placementAPIResource := typeConfig.GetPlacement()
	placementKind := placementAPIResource.Kind
	client, err := util.NewResourceClientFromConfig(kubeConfig, &placementAPIResource)
	if err != nil {
		return err
	}

	expectedClusterNames := []string{}
	for clusterName := range expected {
		expectedClusterNames = append(expectedClusterNames, clusterName)
	}

	return wait.PollImmediate(framework.PollInterval, framework.TestContext.SingleCallTimeout, func() (bool, error) {
		placement, err := client.Resources(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				tl.Errorf("An error occurred while polling for %s %s/%s: %v", placementKind, namespace, name, err)
			}
			return false, nil
		}

		clusterNames, err := util.GetClusterNames(placement)
		if err != nil {
			tl.Errorf("An error occurred while retrieving cluster names for override %s %s/%s: %v", placementKind, namespace, name, err)
			return false, nil
		}
		return !schedulingtypes.PlacementUpdateNeeded(clusterNames, expectedClusterNames), nil
	})
}

func waitForMatchingOverride(tl common.TestLogger, typeConfig typeconfig.Interface, kubeConfig *restclient.Config, name, namespace string, expected32 map[string]int32) error {
	overrideAPIResource := typeConfig.GetOverride()
	overrideKind := overrideAPIResource.Kind
	client, err := util.NewResourceClientFromConfig(kubeConfig, overrideAPIResource)
	if err != nil {
		return err
	}

	expected64 := int32MapToInt64(expected32)

	return wait.PollImmediate(framework.PollInterval, framework.TestContext.SingleCallTimeout, func() (bool, error) {
		override, err := client.Resources(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				tl.Errorf("An error occurred while polling for %s %s/%s: %v", overrideKind, namespace, name, err)
			}
			return false, nil
		}
		return !schedulingtypes.OverrideUpdateNeeded(typeConfig, override, expected64), nil
	})
}

func int32MapToInt64(original map[string]int32) map[string]int64 {
	result := make(map[string]int64)
	for k, v := range original {
		result[k] = int64(v)
	}
	return result
}
