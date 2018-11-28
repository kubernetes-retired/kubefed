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
	"reflect"
	"sort"
	"strings"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	fedschedulingv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/scheduling/v1alpha1"
	clientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/schedulingtypes"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	"github.com/kubernetes-sigs/federation-v2/test/e2e/framework"
	"github.com/kubernetes-sigs/federation-v2/test/e2e/framework/managed"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/runtime"
	restclient "k8s.io/client-go/rest"

	. "github.com/onsi/ginkgo"
)

type FakeScheduleResult struct {
	schedulingtypes.JobScheduleResult
}

// TODO (irfanurrehman): This currently overrides real method only for test
func (r *FakeScheduleResult) PlacementUpdateNeeded(names []string) bool {
	newNames := r.ClusterNames()
	sort.Strings(names)
	sort.Strings(newNames)
	return !reflect.DeepEqual(names, newNames)
}

// TODO (irfanurrehman): This currently overrides real method only for test
func (r *FakeScheduleResult) OverrideUpdateNeeded(typeConfig typeconfig.Interface, obj *unstructured.Unstructured) bool {
	kind := typeConfig.GetOverride().Kind
	qualifiedName := util.NewQualifiedName(obj)

	overrides, err := util.GetClusterOverrides(typeConfig, obj)
	if err != nil {
		wrappedErr := fmt.Errorf("Error reading cluster overrides for %s %q: %v", kind, qualifiedName, err)
		runtime.HandleError(wrappedErr)
		// Updating the overrides should hopefully fix the above problem
		return true
	}

	resultLen := len(r.Result)
	checkLen := 0
	for clusterName, clusterOverrides := range overrides {
		checkInnerLen := 0
		for _, override := range clusterOverrides {
			if strings.Join(override.Path, ".") == schedulingtypes.ParallelismPath {
				value, _ := override.FieldValue.(int64)
				clusterVals, ok := r.Result[clusterName]
				if !ok || value != int64(clusterVals.Parallelism) {
					return true
				}
				checkInnerLen += 1
			}
			if strings.Join(override.Path, ".") == schedulingtypes.CompletionsPath {
				value, _ := override.FieldValue.(int64)
				clusterVals, ok := r.Result[clusterName]
				if !ok || value != int64(clusterVals.Completions) {
					return true
				}
				checkInnerLen += 1
			}
		}
		if checkInnerLen != len(clusterOverrides) {
			return true
		}
		checkLen += 1
	}

	return checkLen != resultLen
}

var _ = Describe("JobSchedulingPreferences", func() {
	f := framework.NewFederationFramework("scheduling")
	tl := framework.NewE2ELogger()

	userAgent := "jsp-test"

	typeConfigs := common.TypeConfigsOrDie(tl)

	schedulingKind := schedulingtypes.JSPKind

	var kubeConfig *restclient.Config
	var fedClient fedclientset.Interface
	var namespace string
	var clusterNames []string

	BeforeEach(func() {
		clusterNames = f.ClusterNames(userAgent)
		if framework.TestContext.TestManagedFederation {
			fixture := managed.NewSchedulingControllerFixture(tl, schedulingKind, f.ControllerConfig(), typeConfigs)
			f.RegisterFixture(fixture)
		} else if framework.TestContext.InMemoryControllers {
			fixture := managed.NewSchedulingManagerFixture(tl, f.ControllerConfig())
			f.RegisterFixture(fixture)
		}
		kubeConfig = f.KubeConfig()
		fedClient = f.FedClient(userAgent)
		namespace = f.TestNamespaceName()
	})

	testCases := map[string]struct {
		totalParallelism    int32
		totalCompletions    int32
		cluster1Parallelism int32
		cluster1Completions int32
		cluster2Parallelism int32
		cluster2Completions int32
	}{
		"Distribution happens evenly across all clusters": {
			totalParallelism:    int32(4),
			totalCompletions:    int32(4),
			cluster1Parallelism: int32(2),
			cluster1Completions: int32(2),
			cluster2Parallelism: int32(2),
			cluster2Completions: int32(2),
		},
	}

	for i := range typeConfigs {
		typeConfig := typeConfigs[i]

		schedulingType := schedulingtypes.GetSchedulingType(typeConfig.GetObjectMeta().Name)
		if schedulingType == nil || schedulingType.Kind != schedulingKind {
			continue
		}

		templateKind := typeConfig.GetTemplate().Kind

		Describe(fmt.Sprintf("scheduling for %s", templateKind), func() {
			for testName, tc := range testCases {
				It(fmt.Sprintf("should result in %s", testName), func() {
					clusterCount := len(clusterNames)
					if clusterCount != 2 {
						framework.Skipf("Tests of JobSchedulingPreferences requires 2 clusters but got: %d", clusterCount)
					}

					var jspSpec fedschedulingv1a1.JobSchedulingPreferenceSpec
					jspSpec = jspSpecWithoutClusterWeights(tc.totalParallelism, tc.totalCompletions)

					expected := &FakeScheduleResult{}
					expected.Result = map[string]schedulingtypes.ClusterJobValues{
						clusterNames[0]: {
							Completions: tc.cluster1Completions,
							Parallelism: tc.cluster1Parallelism,
						},
						clusterNames[1]: {
							Completions: tc.cluster1Completions,
							Parallelism: tc.cluster1Parallelism,
						},
					}

					name, err := createJSPTestObjs(fedClient, typeConfig, kubeConfig, jspSpec, namespace)
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
		// This test runs only for one type/kind
		break
	}
})

func jspSpecWithoutClusterWeights(parallelism, completions int32) fedschedulingv1a1.JobSchedulingPreferenceSpec {
	return fedschedulingv1a1.JobSchedulingPreferenceSpec{
		TotalParallelism: parallelism,
		TotalCompletions: completions,
	}
}

func createJSPTestObjs(fedClient clientset.Interface, typeConfig typeconfig.Interface, kubeConfig *restclient.Config, jspSpec fedschedulingv1a1.JobSchedulingPreferenceSpec, namespace string) (string, error) {
	name, err := createTemplate(typeConfig, kubeConfig, namespace)
	if err != nil {
		return "", err
	}
	jsp := &fedschedulingv1a1.JobSchedulingPreference{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: jspSpec,
	}
	_, err = fedClient.SchedulingV1alpha1().JobSchedulingPreferences(namespace).Create(jsp)
	if err != nil {
		return "", err
	}

	return name, nil
}
