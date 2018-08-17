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
	"reflect"
	"testing"

	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	fedschedulingv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/scheduling/v1alpha1"
	clientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	"github.com/kubernetes-sigs/federation-v2/test/integration/framework"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	restclient "k8s.io/client-go/rest"
)

const (
	federatedDeployment = "FederatedDeployment"
	federatedReplicaSet = "FederatedReplicaSet"
)

// TestReplicaSchedulingPreference validates basic replica scheduling preference calculations.
var TestReplicaSchedulingPreference = func(t *testing.T) {
	t.Parallel()
	tl := framework.NewIntegrationLogger(t)

	controllerFixture, fedClient := initRSPTest(tl, FedFixture)
	defer controllerFixture.TearDown(tl)

	clusters := getClusterNames(fedClient, FedFixture.SystemNamespace)
	if len(clusters) != 2 {
		tl.Fatalf("Expected two clusters to be part of Federation Fixture setup")
	}

	kubeClient := FedFixture.KubeApi.NewClient(tl, "rsp")
	testNamespace := framework.CreateTestNamespace(tl, kubeClient, "rsp-")

	targetKinds := []string{
		federatedDeployment,
		federatedReplicaSet,
	}

	for _, targetKind := range targetKinds {
		testCases := map[string]struct {
			rspSpec  fedschedulingv1a1.ReplicaSchedulingPreferenceSpec
			expected map[string]int32
		}{
			"Replicas spread equally in clusters, with no explicit per cluster preferences": {
				rspSpec: rspSpecWithoutClusterList(int32(4), targetKind),
				expected: map[string]int32{
					clusters[0]: int32(2),
					clusters[1]: int32(2),
				},
			},
			"Replicas spread in proportion of weights when explicit preferences with weights specified": {
				rspSpec: rspSpecWithClusterList(int32(6), int64(2), int64(1), int64(0), int64(0), clusters, targetKind),
				expected: map[string]int32{
					clusters[0]: int32(4),
					clusters[1]: int32(2),
				},
			},
			"Replicas spread considering min replicas when both minreplica and weights specified": {
				rspSpec: rspSpecWithClusterList(int32(6), int64(2), int64(1), int64(3), int64(3), clusters, targetKind),
				expected: map[string]int32{
					clusters[0]: int32(3),
					clusters[1]: int32(3),
				},
			},
		}

		for testName, testCase := range testCases {
			t.Run(testName, func(t *testing.T) {
				name, err := createTestObjs(testCase.rspSpec, testNamespace, targetKind, fedClient)
				if err != nil {
					tl.Fatalf("Creation of test objects failed in federation")
				}

				err = waitForMatchingPlacement(fedClient, name, testNamespace, targetKind, testCase.expected)
				if err != nil {
					tl.Fatalf("Failed waiting for matching placements")
				}

				err = waitForMatchingOverride(fedClient, name, testNamespace, targetKind, testCase.expected)
				if err != nil {
					tl.Fatalf("Failed waiting for matching overrides")
				}
			})
		}
	}
}

func initRSPTest(tl common.TestLogger, fedFixture *framework.FederationFixture) (*framework.ControllerFixture, clientset.Interface) {
	config := fedFixture.KubeApi.NewConfig(tl)
	fixture := framework.NewRSPControllerFixture(tl, config, fedFixture.SystemNamespace, fedFixture.SystemNamespace, metav1.NamespaceAll)
	restclient.AddUserAgent(config, "rsp-test")
	client := clientset.NewForConfigOrDie(config)

	return fixture, client
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
func createTestObjs(rspSpec fedschedulingv1a1.ReplicaSchedulingPreferenceSpec, namespace, targetKind string, fedClient clientset.Interface) (string, error) {
	replicas := int32(1)
	name := ""
	var wrapErr error

	switch targetKind {
	case federatedDeployment:
		t, err := fedClient.CoreV1alpha1().FederatedDeployments(namespace).Create(getFederatedDeploymentTemplate(namespace, replicas).(*fedv1a1.FederatedDeployment))
		name = t.Name
		wrapErr = err
	case federatedReplicaSet:
		t, err := fedClient.CoreV1alpha1().FederatedReplicaSets(namespace).Create(getFederatedReplicaSetTemplate(namespace, replicas).(*fedv1a1.FederatedReplicaSet))
		name = t.Name
		wrapErr = err
	}

	if wrapErr != nil {
		return "", wrapErr
	}

	rsp := &fedschedulingv1a1.ReplicaSchedulingPreference{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: rspSpec,
	}
	_, err := fedClient.SchedulingV1alpha1().ReplicaSchedulingPreferences(namespace).Create(rsp)
	if err != nil {
		return "", err
	}

	return name, nil
}

func waitForMatchingPlacement(fedClient clientset.Interface, name, namespace, targetKind string, expected map[string]int32) error {
	err := wait.Poll(framework.DefaultWaitInterval, wait.ForeverTestTimeout, func() (bool, error) {
		var wrapErr error
		clusterNames := []string{}
		switch targetKind {
		case federatedDeployment:
			p, err := fedClient.CoreV1alpha1().FederatedDeploymentPlacements(namespace).Get(name, metav1.GetOptions{})
			clusterNames = p.Spec.ClusterNames
			wrapErr = err
		case federatedReplicaSet:
			p, err := fedClient.CoreV1alpha1().FederatedReplicaSetPlacements(namespace).Get(name, metav1.GetOptions{})
			clusterNames = p.Spec.ClusterNames
			wrapErr = err
		}
		if wrapErr != nil {
			return false, nil
		}

		if len(clusterNames) > 0 {
			totalClusters := 0
			for _, clusterName := range clusterNames {
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

func waitForMatchingOverride(fedClient clientset.Interface, name, namespace, targetKind string, expected map[string]int32) error {
	err := wait.Poll(framework.DefaultWaitInterval, wait.ForeverTestTimeout, func() (bool, error) {
		var override pkgruntime.Object
		var wrapErr error
		switch targetKind {
		case federatedDeployment:
			override, wrapErr = fedClient.CoreV1alpha1().FederatedDeploymentOverrides(namespace).Get(name, metav1.GetOptions{})
		case federatedReplicaSet:
			override, wrapErr = fedClient.CoreV1alpha1().FederatedReplicaSetOverrides(namespace).Get(name, metav1.GetOptions{})
		}
		if wrapErr != nil {
			return false, nil
		}

		if override != nil {
			// We do not consider a case where overrides won't have any clusters listed
			match := false
			totalClusters := 0
			overrides := reflect.ValueOf(override).Elem().FieldByName("Spec").FieldByName("Overrides")

			for i := 0; i < overrides.Len(); i++ {
				o := overrides.Index(i)
				name := o.FieldByName("ClusterName").String()
				specReplicas := o.FieldByName("Replicas").Elem().Int()

				match = false // Check for each cluster listed in overrides
				totalClusters++
				replicas, exists := expected[name]
				// Overrides should have exact mapping replicas as in expected
				if !exists {
					return false, nil
				}
				if int32(specReplicas) == replicas {
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

func getClusterNames(fedClient clientset.Interface, fedNamespace string) []string {
	clusters := []string{}

	clusterList, err := fedClient.CoreV1alpha1().FederatedClusters(fedNamespace).List(metav1.ListOptions{})
	if err != nil || clusterList == nil {
		return clusters
	}
	for _, cluster := range clusterList.Items {
		clusters = append(clusters, cluster.Name)
	}

	return clusters
}

func getFederatedDeploymentTemplate(namespace string, replicas int32) pkgruntime.Object {
	return &fedv1a1.FederatedDeployment{
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
}

func getFederatedReplicaSetTemplate(namespace string, replicas int32) pkgruntime.Object {
	return &fedv1a1.FederatedReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-replicaset-",
			Namespace:    namespace,
		},
		Spec: fedv1a1.FederatedReplicaSetSpec{
			Template: appsv1.ReplicaSet{
				Spec: appsv1.ReplicaSetSpec{
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
}
