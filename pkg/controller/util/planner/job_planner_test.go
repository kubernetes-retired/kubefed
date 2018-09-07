/*
Copyright 2016 The Kubernetes Authors.

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

package planner

import (
	"testing"

	fedschedulingv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/scheduling/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func doJSPCheck(t *testing.T, weights map[string]int32, toDistribute int32, clusters []string, expected map[string]int32) {
	planer := NewJobPlanner(&fedschedulingv1a1.JobSchedulingPreference{
		Spec: fedschedulingv1a1.JobSchedulingPreferenceSpec{
			ClusterWeights:   weights,
			TotalParallelism: int32(toDistribute),
		},
	})
	plan := planer.Plan(clusters, toDistribute, "")
	assert.EqualValues(t, expected, plan)
}

func TestJobPlanner(t *testing.T) {
	doJSPCheck(t, map[string]int32{
		AllClusters: 1},
		50, []string{"A", "B", "C"},
		// hash dependent
		map[string]int32{"A": 16, "B": 17, "C": 17})

	doJSPCheck(t, map[string]int32{
		AllClusters: 1},
		50, []string{"A", "B"},
		map[string]int32{"A": 25, "B": 25})

	doJSPCheck(t, map[string]int32{},
		50, []string{"A", "B"},
		map[string]int32{"A": 25, "B": 25})

	doJSPCheck(t, map[string]int32{
		AllClusters: 1},
		1, []string{"A", "B"},
		// hash dependent; We do not create a job in clusters
		// which get parallelism as 0
		map[string]int32{"B": 1})

	doJSPCheck(t, map[string]int32{
		AllClusters: 1},
		1, []string{"A"},
		map[string]int32{"A": 1})

	doJSPCheck(t, map[string]int32{
		AllClusters: 1},
		1, []string{},
		map[string]int32{})

	doJSPCheck(t, map[string]int32{
		"A": 1,
		"B": 2,
	},
		10, []string{"A", "B"},
		map[string]int32{"A": 3, "B": 7})

	doJSPCheck(t, map[string]int32{
		"A": 1,
		"B": 2,
	},
		10, []string{"A", "B", "C"},
		map[string]int32{"A": 3, "B": 7})
}
