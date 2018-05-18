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

package integration

import (
	"testing"

	fedschedulingv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/federatedscheduling/v1alpha1"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/federation/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/client/clientset_generated/clientset"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	"github.com/kubernetes-sigs/federation-v2/test/integration/framework"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	defaultTestNS = "fed-test-ns"
)

// TestReplicaSchedulingPreference validates basic replica scheduling preference calculations.
func TestReplicaSchedulingPreference(t *testing.T) {
	tl := framework.NewIntegrationLogger(t)
	fedFixture := framework.SetUpFederationFixture(tl, 2)
	defer fedFixture.TearDown(tl)

	controllerFixture, fedClient := initRSPTest(tl, fedFixture)
	defer controllerFixture.TearDown(tl)

	clusters := getClusterNames(fedClient)
	if len(clusters) != 2 {
		tl.Fatalf("Expected two clusters to be part of Federation Fixture setup")
	}

	testCases := map[string]struct {
		prefSpec fedschedulingv1a1.ReplicaSchedulingPreferenceSpec
		expected map[string]int32
	}{
		"Replicas spread equally in clusters, with no explicit per cluster preferences": {
			prefSpec: prefSpecWithoutClusterList(int32(4)),
			expected: map[string]int32{
				clusters[0]: int32(2),
				clusters[1]: int32(2),
			},
		},
		"Replicas spread in proportion of weights when explicit preferences with weights specified": {
			prefSpec: prefSpecWithClusterList(int32(6), int64(2), int64(1), int64(0), int64(0), clusters),
			expected: map[string]int32{
				clusters[0]: int32(4),
				clusters[1]: int32(2),
			},
		},
		"Replicas spread considering min replicas when both minreplica and weights specified": {
			prefSpec: prefSpecWithClusterList(int32(6), int64(2), int64(1), int64(3), int64(3), clusters),
			expected: map[string]int32{
				clusters[0]: int32(3),
				clusters[1]: int32(3),
			},
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			name, err := createTestObjs(testCase.prefSpec, defaultTestNS, fedClient)
			if err != nil {
				tl.Fatalf("Creation of test objects failed in federation")
			}

			err = waitForMatchingPlacement(fedClient, name, defaultTestNS, testCase.expected)
			if err != nil {
				tl.Fatalf("Failed waiting for matching placements")
			}

			err = waitForMatchingOverride(fedClient, name, defaultTestNS, testCase.expected)
			if err != nil {
				tl.Fatalf("Failed waiting for matching overrides")
			}
		})
	}
}

func initRSPTest(tl common.TestLogger, fedFixture *framework.FederationFixture) (*framework.ControllerFixture, clientset.Interface) {
	fedConfig := fedFixture.FedApi.NewConfig(tl)
	kubeConfig := fedFixture.KubeApi.NewConfig(tl)
	crConfig := fedFixture.CrApi.NewConfig(tl)
	fixture := framework.NewRSPControllerFixture(tl, fedConfig, kubeConfig, crConfig)
	client := fedFixture.FedApi.NewClient(tl, "rsp-test")

	return fixture, client
}

func prefSpecWithoutClusterList(total int32) fedschedulingv1a1.ReplicaSchedulingPreferenceSpec {
	return fedschedulingv1a1.ReplicaSchedulingPreferenceSpec{
		TotalReplicas: total,
		//TODO: TargetRef is actually unused in this pass of implementation
		PreferenceTargetRef: fedschedulingv1a1.ObjectReference{},
		Clusters:            map[string]fedschedulingv1a1.ClusterPreferences{},
	}
}

// This assumes test setup using 2 clusters
func prefSpecWithClusterList(total int32, w1, w2, min1, min2 int64, clusters []string) fedschedulingv1a1.ReplicaSchedulingPreferenceSpec {
	prefSpec := prefSpecWithoutClusterList(total)
	prefSpec.Clusters = map[string]fedschedulingv1a1.ClusterPreferences{
		clusters[0]: {
			MinReplicas: min1,
			Weight:      w1,
		},
		clusters[1]: {
			MinReplicas: min2,
			Weight:      w2,
		},
	}

	return prefSpec
}
func createTestObjs(prefSpec fedschedulingv1a1.ReplicaSchedulingPreferenceSpec, namespace string, fedClient clientset.Interface) (string, error) {
	replicas := int32(1)
	template := &fedv1a1.FederatedDeployment{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-deployment-",
			Namespace:    namespace,
		},
		Spec: fedv1a1.FederatedDeploymentSpec{
			Template: appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Replicas: &replicas,
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"foo": "bar"},
					},
					Template: apiv1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"foo": "bar"},
						},
						Spec: apiv1.PodSpec{
							Containers: []apiv1.Container{
								{
									Name:  "nginx",
									Image: "nginx",
								},
							},
						},
					},
				},
			},
		},
	}
	t, err := fedClient.FederationV1alpha1().FederatedDeployments(namespace).Create(template)
	if err != nil {
		return "", err
	}

	rsp := &fedschedulingv1a1.ReplicaSchedulingPreference{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.Name,
			Namespace: namespace,
		},
		Spec: prefSpec,
	}
	_, err = fedClient.FederatedschedulingV1alpha1().ReplicaSchedulingPreferences(namespace).Create(rsp)
	if err != nil {
		return "", err
	}

	return t.Name, nil
}

func waitForMatchingPlacement(fedClient clientset.Interface, name, namespace string, expected map[string]int32) error {
	err := wait.Poll(framework.DefaultWaitInterval, wait.ForeverTestTimeout, func() (bool, error) {
		placement, err := fedClient.FederationV1alpha1().FederatedDeploymentPlacements(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		if placement != nil {
			totalClusters := 0
			for _, clusterName := range placement.Spec.ClusterNames {
				totalClusters++
				_, exists := expected[clusterName]
				if !exists {
					return false, nil
				}
			}

			// All clusters in placement has a matched cluster name as in expected.
			if totalClusters == len(expected) {
				return true, nil
			}
		}
		return false, nil
	})

	return err
}

func waitForMatchingOverride(fedClient clientset.Interface, name, namespace string, expected map[string]int32) error {
	err := wait.Poll(framework.DefaultWaitInterval, wait.ForeverTestTimeout, func() (bool, error) {
		override, err := fedClient.FederationV1alpha1().FederatedDeploymentOverrides(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		if override != nil {
			// We do not consider a case where overrides won't have any clusters listed
			match := false
			totalClusters := 0
			for _, override := range override.Spec.Overrides {
				match = false // Check for each cluster listed in overrides
				totalClusters++
				replicas, exists := expected[override.ClusterName]
				// Overrides should have exact mapping replicas as in expected
				if !exists {
					return false, nil
				}
				if override.Replicas != nil && *override.Replicas == replicas {
					match = true
					continue
				}
			}

			if match && (totalClusters == len(expected)) {
				return true, nil
			}
		}
		return false, nil
	})

	return err
}

func getClusterNames(fedClient clientset.Interface) []string {
	clusters := []string{}

	clusterList, err := fedClient.Federation().FederatedClusters().List(metav1.ListOptions{})
	if err != nil || clusterList == nil {
		return clusters
	}
	for _, cluster := range clusterList.Items {
		clusters = append(clusters, cluster.Name)
	}

	return clusters
}
