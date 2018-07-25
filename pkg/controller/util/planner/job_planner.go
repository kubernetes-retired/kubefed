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

package planner

import (
	"hash/fnv"
	"sort"

	fedschedulingv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/scheduling/v1alpha1"
)

// JobPlanner decides how should to distribute jobs into federated clusters,
// based on the details provided in JobSchedulingPreference.
type JobPlanner struct {
	preferences *fedschedulingv1a1.JobSchedulingPreference
}

type clusterWithWeight struct {
	clusterName string
	hash        uint32
	Weight      int32
}

type ClustersByWeight []*clusterWithWeight

func (a ClustersByWeight) Len() int      { return len(a) }
func (a ClustersByWeight) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// Preferences are sorted according by decreasing weight and increasing hash (built on top of cluster name and rs name).
// Sorting is made by a hash to avoid assigning single-replica rs to the alphabetically smallest cluster.
func (a ClustersByWeight) Less(i, j int) bool {
	return (a[i].Weight > a[j].Weight) || (a[i].Weight == a[j].Weight && a[i].hash < a[j].hash)
}

func NewJobPlanner(preferences *fedschedulingv1a1.JobSchedulingPreference) *JobPlanner {
	return &JobPlanner{
		preferences: preferences,
	}
}

// This assumes that there is at least one available cluster and distributes
// the parallelism or completions of the job equally among clusters or in
// the ratio of weights if specified.
func (p *JobPlanner) Plan(availableClusters []string, toDistribute int32, hashKey string) map[string]int32 {

	named := func(name string, weight int32) *clusterWithWeight {
		// Seems to work better than addler for our case.
		hasher := fnv.New32()
		hasher.Write([]byte(name))
		hasher.Write([]byte(hashKey))

		return &clusterWithWeight{
			clusterName: name,
			hash:        hasher.Sum32(),
			Weight:      weight,
		}
	}

	numClusters := len(availableClusters)
	plan := make(map[string]int32, numClusters)
	if numClusters == 0 {
		return plan
	}
	// weightedList will only store those cluster values which have
	// weights specified in the preferences.
	weightedList := make([]*clusterWithWeight, 0, numClusters)
	weightSum := int32(0)

	for _, cluster := range availableClusters {
		if weight, found := p.preferences.Spec.ClusterWeights[cluster]; found {
			weightedList = append(weightedList, named(cluster, weight))
			weightSum += weight
		} else {
			if weight, found := p.preferences.Spec.ClusterWeights[AllClusters]; found {
				weightedList = append(weightedList, named(cluster, weight))
				weightSum += weight
			} else {
				// Clusters which do not have explicit weights given, while
				// some others have, are treated equivalent to 0 weight.
				plan[cluster] = int32(0)
			}
		}
	}
	sort.Sort(ClustersByWeight(weightedList))

	remaining := toDistribute
	if weightSum > 0 {
		for _, cluster := range weightedList {
			if remaining > 0 {
				// Distribute the numbers as per the weight ratio, rounding fractions always up.
				distributed := (toDistribute*cluster.Weight + weightSum - 1) / weightSum
				distributed = minInt32(distributed, remaining)
				plan[cluster.clusterName] = distributed
				remaining -= distributed
			}
		}
	} else {
		// No weights, default distribution
		distributionStep := toDistribute / int32(numClusters)
		if distributionStep > 0 {
			for _, clusterName := range availableClusters {
				plan[clusterName] = distributionStep
			}
		}

		remainder := toDistribute % int32(numClusters)
		if remainder > 0 {
			for _, clusterName := range availableClusters {
				plan[clusterName] += 1
				remainder--
				if remainder == 0 {
					break
				}
			}
		}
	}

	return plan
}

func minInt32(a int32, b int32) int32 {
	if a < b {
		return a
	}
	return b
}
