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

package placement

import (
	"fmt"

	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
)

// namespacedPlacementPlugin determines placement for namespaced
// resources (e.g. ConfigMap).
//
// If federation is deployed cluster-wide, placement is the
// intersection of the placement for the resource and the placement of
// the namespace containing the resource.
//
// If federation is limited to a single namespace, placement is
// determined as the intersection of resource and namespace placement
// if namespace placement exists.  If namespace placement does not
// exist, resource placement will be used verbatim.  This is possible
// because the single namespace by definition must exist on member
// clusters, so namespace placement becomes a mechanism for limiting
// rather than allowing propagation.
type namespacedPlacementPlugin struct {
	targetNamespace string
	resourcePlugin  PlacementPlugin
	namespacePlugin *ResourcePlacementPlugin
}

func NewNamespacedPlacementPlugin(resourceClient, namespaceClient util.ResourceClient, targetNamespace string, resourceTriggerFunc, namespaceTriggerFunc func(pkgruntime.Object)) PlacementPlugin {
	return &namespacedPlacementPlugin{
		targetNamespace: targetNamespace,
		resourcePlugin:  NewResourcePlacementPlugin(resourceClient, targetNamespace, resourceTriggerFunc),
		namespacePlugin: newResourcePlacementPluginWithOk(namespaceClient, targetNamespace, namespaceTriggerFunc),
	}
}

func (p *namespacedPlacementPlugin) Run(stopCh <-chan struct{}) {
	go p.resourcePlugin.Run(stopCh)
	go p.namespacePlugin.Run(stopCh)
}

func (p *namespacedPlacementPlugin) HasSynced() bool {
	return p.resourcePlugin.HasSynced() && p.namespacePlugin.HasSynced()
}

func (p *namespacedPlacementPlugin) ComputePlacement(qualifiedName util.QualifiedName, clusters []*fedv1a1.FederatedCluster) (selectedClusters, unselectedClusters []string, err error) {
	selectedClusters, unselectedClusters, err = p.resourcePlugin.ComputePlacement(qualifiedName, clusters)
	if err != nil {
		return nil, nil, err
	}

	placementName := util.QualifiedName{Namespace: qualifiedName.Namespace, Name: qualifiedName.Namespace}
	namespaceSelectedClusters, _, ok, err := p.namespacePlugin.computePlacementWithOk(placementName, clusters)
	if err != nil {
		return nil, nil, err
	}
	if !ok && p.targetNamespace != metav1.NamespaceAll {
		// Use the resource placement if no namespace placement is
		// available and federation is targeting a single namespace.
		return selectedClusters, unselectedClusters, nil
	}

	resourceClusterSet := sets.NewString(selectedClusters...)
	namespaceClusterSet := sets.NewString(namespaceSelectedClusters...)
	clusterSet := sets.NewString(getClusterNames(clusters)...)

	// If both namespace and resource placement exist, the desired
	// list of clusters should be their intersection.
	selectedSet := resourceClusterSet.Intersection(namespaceClusterSet)

	return selectedSet.List(), clusterSet.Difference(selectedSet).List(), nil
}

func (p *namespacedPlacementPlugin) GetPlacement(key string) (*unstructured.Unstructured, error) {
	return nil, fmt.Errorf("Not implemented")
}
