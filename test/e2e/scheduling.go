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
	"time"

	"github.com/pkg/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	restclient "k8s.io/client-go/rest"

	"sigs.k8s.io/kubefed/pkg/apis/core/typeconfig"
	fedv1b1 "sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	fedschedulingv1a1 "sigs.k8s.io/kubefed/pkg/apis/scheduling/v1alpha1"
	genericclient "sigs.k8s.io/kubefed/pkg/client/generic"
	"sigs.k8s.io/kubefed/pkg/controller/util"
	"sigs.k8s.io/kubefed/pkg/schedulingtypes"
	"sigs.k8s.io/kubefed/test/common"
	"sigs.k8s.io/kubefed/test/e2e/framework"

	. "github.com/onsi/ginkgo" //nolint:stylecheck
)

var _ = Describe("Scheduling", func() {
	f := framework.NewKubeFedFramework("scheduling")
	tl := framework.NewE2ELogger()

	userAgent := "rsp-test"

	schedulingTypes := GetSchedulingTypes(tl)

	var kubeConfig *restclient.Config
	var genericClient genericclient.Client
	var namespace string
	var clusterNames []string
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
				typeConfig, err := common.GetTypeConfig(client, targetTypeName, f.KubeFedSystemNamespace())
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

		controllerFixture, _ := framework.NewSchedulingManagerFixture(tl, f.ControllerConfig())
		f.RegisterFixture(controllerFixture)
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
			intersection  bool
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
			"target clusters are the intersection of the RSP clusters and federated resource placement": {
				intersection: true,
				total:        int32(6),
				weight1:      int64(2),
				weight2:      int64(1),
				min1:         int64(3),
				min2:         int64(3),
				cluster1:     int32(6),
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

						if tc.intersection {
							testNs := f.EnsureTestFederatedNamespace(true)
							fedNs := f.KubeFedSystemNamespace()
							createIntersectionEnvironment(tl, genericClient, fedNs, clusterNames[0])

							rspSpec.IntersectWithClusterSelector = true
							expected = map[string]int32{
								clusterNames[0]: tc.cluster1,
							}

							defer func() {
								destroyIntersectionEnvironment(tl, genericClient, testNs, fedNs, clusterNames[0])
							}()
						}

						name, err := createTestObjs(tl, genericClient, typeConfig, kubeConfig, rspSpec, namespace)
						if err != nil {
							tl.Fatalf("Creation of test objects in the host cluster failed: %v", err)
						}

						err = waitForMatchingFederatedObject(tl, typeConfig, kubeConfig, name, namespace, expected)
						if err != nil {
							tl.Fatalf("Failed waiting for matching federated object: %v", err)
						}

						err = deleteTestObj(typeConfig, kubeConfig, name, namespace)
						if err != nil {
							tl.Fatalf("Deletion of a test object from the host cluster failed: %v", err)
						}
					})
				}
			})
		}
	})
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

	if rspSpec.IntersectWithClusterSelector {
		clusterSelector := map[string]string{
			"foo": "bar",
		}

		err = util.SetClusterSelector(fedObject, clusterSelector)
		if err != nil {
			return "", err
		}
	}

	createdFedObject, err := federatedTypeClient.Resources(namespace).Create(context.Background(), fedObject, metav1.CreateOptions{})
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

	err = federatedTypeClient.Resources(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
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
		fedObject, err := client.Resources(namespace).Get(context.Background(), name, metav1.GetOptions{})
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

func createIntersectionEnvironment(tl common.TestLogger, client genericclient.Client, kubefedNamespace string, clusterName string) {
	updateClusterLabel(tl, client, kubefedNamespace, clusterName, true)
}

func destroyIntersectionEnvironment(tl common.TestLogger, client genericclient.Client, testNamespace *unstructured.Unstructured, kubefedNamespace string, clusterName string) {
	testNamespaceKey := util.NewQualifiedName(testNamespace).String()
	err := client.Delete(context.Background(), testNamespace, testNamespace.GetNamespace(), testNamespace.GetName())
	if err != nil && !apierrors.IsNotFound(err) {
		tl.Fatalf("Error deleting FederatedNamespace %q: %v", testNamespaceKey, err)
	}

	updateClusterLabel(tl, client, kubefedNamespace, clusterName, false)
}

func updateClusterLabel(tl common.TestLogger, client genericclient.Client, kubefedNamespace string, clusterName string, addTestLabel bool) {
	fedCluster := &unstructured.Unstructured{}
	fedCluster.SetGroupVersionKind(schema.GroupVersionKind{
		Kind:    "KubeFedCluster",
		Group:   fedv1b1.SchemeGroupVersion.Group,
		Version: fedv1b1.SchemeGroupVersion.Version,
	})
	// We retry couple of times on conflict
	err := wait.PollImmediate(1*time.Second, 10*time.Second, func() (bool, error) {
		err := client.Get(context.Background(), fedCluster, kubefedNamespace, clusterName)
		if err != nil {
			tl.Fatalf("Cannot get KubeFedCluster %q from namespace %q: %v", clusterName, kubefedNamespace, err)
		}

		if addTestLabel {
			addLabel(fedCluster, "foo", "bar")
		} else {
			removeLabel(fedCluster, "foo", "bar")
		}
		err = client.Update(context.TODO(), fedCluster)
		if err == nil {
			return true, nil
		}
		if apierrors.IsConflict(err) {
			tl.Logf("Got conflit updating label %q (add=%t) to KubeFedCluster %q", "foo:bar. Will Retry.", addTestLabel, clusterName)
			return false, nil
		}
		return false, errors.Wrapf(err, "failed to update resource")
	})
	if err != nil {
		tl.Fatalf("Error updating label %q (add=%t) to KubeFedCluster %q: %v", "foo:bar", addTestLabel, clusterName, err)
	}
}

func int32MapToInt64(original map[string]int32) map[string]int64 {
	result := make(map[string]int64)
	for k, v := range original {
		result[k] = int64(v)
	}
	return result
}

func addLabel(obj *unstructured.Unstructured, key, value string) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	labels[key] = value
	obj.SetLabels(labels)
}

func removeLabel(obj *unstructured.Unstructured, key, value string) {
	labels := obj.GetLabels()
	if labels == nil || labels[key] != value {
		return
	}
	delete(labels, key)
	obj.SetLabels(labels)
}
