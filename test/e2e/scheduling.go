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
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
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

	typeConfigs := common.TypeConfigsOrDie(tl)

	schedulingKind := schedulingtypes.RSPKind

	var kubeConfig *restclient.Config
	var fedClient fedclientset.Interface
	var namespace string
	var clusterNames []string

	BeforeEach(func() {
		clusterNames = f.ClusterNames(userAgent)
		if framework.TestContext.TestManagedFederation {
			fixture := managed.NewRSPControllerFixture(tl, f.ControllerConfig(), typeConfigs)
			f.RegisterFixture(fixture)
		} else if framework.TestContext.InMemoryControllers {
			fixture := managed.NewSchedulerControllerFixture(tl, f.ControllerConfig())
			f.RegisterFixture(fixture)
		}
		kubeConfig = f.KubeConfig()
		fedClient = f.FedClient(userAgent)
		namespace = f.TestNamespaceName()
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

	for i := range typeConfigs {
		typeConfig := typeConfigs[i]

		schedulingType := schedulingtypes.GetSchedulingType(typeConfig.GetObjectMeta().Name)
		if schedulingType == nil || schedulingType.Kind != schedulingKind {
			continue
		}

		// TODO(marun) Rename RSP field s/TargetKind/TemplateKind/
		templateKind := typeConfig.GetTemplate().Kind

		Describe(fmt.Sprintf("scheduling for %s", templateKind), func() {
			for testName, tc := range testCases {
				It(fmt.Sprintf("should result in %s", testName), func() {
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

					expected := schedulingtypes.NewReplicaScheduleResult(map[string]int64{
						clusterNames[0]: int64(tc.cluster1),
						clusterNames[1]: int64(tc.cluster2),
					})

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
	template, err := common.NewTestTemplate(typeConfig, namespace)
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

func waitForMatchingPlacement(tl common.TestLogger, typeConfig typeconfig.Interface, kubeConfig *restclient.Config, name, namespace string, expected *schedulingtypes.ReplicaScheduleResult) error {
	placementAPIResource := typeConfig.GetPlacement()
	placementKind := placementAPIResource.Kind
	client, err := util.NewResourceClientFromConfig(kubeConfig, &placementAPIResource)
	if err != nil {
		return err
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
		return !expected.PlacementUpdateNeeded(clusterNames), nil
	})
}

func waitForMatchingOverride(tl common.TestLogger, typeConfig typeconfig.Interface, kubeConfig *restclient.Config, name, namespace string, expected *schedulingtypes.ReplicaScheduleResult) error {
	overrideAPIResource := typeConfig.GetOverride()
	overrideKind := overrideAPIResource.Kind
	client, err := util.NewResourceClientFromConfig(kubeConfig, overrideAPIResource)
	if err != nil {
		return err
	}

	return wait.PollImmediate(framework.PollInterval, framework.TestContext.SingleCallTimeout, func() (bool, error) {
		override, err := client.Resources(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				tl.Errorf("An error occurred while polling for %s %s/%s: %v", overrideKind, namespace, name, err)
			}
			return false, nil
		}
		return !expected.OverrideUpdateNeeded(typeConfig, override), nil
	})
}
