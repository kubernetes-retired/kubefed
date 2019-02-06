/*
Copyright 2019 The Kubernetes Authors.

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

package sync

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"

	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
)

// computeNamespacedPlacement determines placement for namespaced
// federated resources (e.g. FederatedConfigMap).
//
// If federation is deployed cluster-wide, placement is the
// intersection of the placement for the federated resource and the
// placement of the federated namespace containing the resource.
//
// If federation is limited to a single namespace, placement is
// determined as the intersection of resource and namespace placement
// if namespace placement exists.  If namespace placement does not
// exist, resource placement will be used verbatim.  This is possible
// because the single namespace by definition must exist on member
// clusters, so namespace placement becomes a mechanism for limiting
// rather than allowing propagation.
func computeNamespacedPlacement(resource, namespace *unstructured.Unstructured, clusters []*fedv1a1.FederatedCluster, limitedScope bool) (selectedClusters, unselectedClusters []string, err error) {
	selectedClusters, unselectedClusters, err = computePlacement(resource, clusters)
	if err != nil {
		return nil, nil, err
	}

	clusterNames := getClusterNames(clusters)

	if namespace == nil {
		if limitedScope {
			// Use the resource placement verbatim if no federated
			// namespace is present and federation is targeting a
			// single namespace.
			return selectedClusters, unselectedClusters, nil
		}
		// Resource should not exist in any member clusters.
		return []string{}, clusterNames, nil
	}

	namespaceSelectedClusters, _, err := computePlacement(namespace, clusters)
	if err != nil {
		return nil, nil, err
	}

	resourceClusterSet := sets.NewString(selectedClusters...)
	namespaceClusterSet := sets.NewString(namespaceSelectedClusters...)
	clusterSet := sets.NewString(clusterNames...)

	// If both namespace and resource placement exist, the desired
	// list of clusters is their intersection.
	selectedSet := resourceClusterSet.Intersection(namespaceClusterSet)

	return selectedSet.List(), clusterSet.Difference(selectedSet).List(), nil
}

// computePlacement determines the selected and unselected clusters
// for a federated resource.
func computePlacement(resource *unstructured.Unstructured, clusters []*fedv1a1.FederatedCluster) (selectedClusters, unselectedClusters []string, err error) {
	selectedNames, err := selectedClusterNames(resource, clusters)
	if err != nil {
		return nil, nil, err
	}
	clusterNames := getClusterNames(clusters)
	clusterSet := sets.NewString(clusterNames...)
	selectedSet := sets.NewString(selectedNames...)
	return clusterSet.Intersection(selectedSet).List(), clusterSet.Difference(selectedSet).List(), nil
}

func selectedClusterNames(resource *unstructured.Unstructured, clusters []*fedv1a1.FederatedCluster) ([]string, error) {
	directive, err := util.GetPlacementDirective(resource)
	if err != nil {
		return nil, err
	}

	if directive.ClusterNames != nil {
		return directive.ClusterNames, nil
	}

	selectedNames := []string{}
	for _, cluster := range clusters {
		if directive.ClusterSelector.Matches(labels.Set(cluster.Labels)) {
			selectedNames = append(selectedNames, cluster.Name)
		}
	}
	return selectedNames, nil
}

func getClusterNames(clusters []*fedv1a1.FederatedCluster) []string {
	clusterNames := []string{}
	for _, cluster := range clusters {
		clusterNames = append(clusterNames, cluster.Name)
	}
	return clusterNames
}
