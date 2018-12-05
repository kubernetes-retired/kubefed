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
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
)

type namespacedPlacementPlugin struct {
	resourcePlugin  PlacementPlugin
	namespacePlugin *ResourcePlacementPlugin
}

func NewNamespacedPlacementPlugin(resourceClient, namespaceClient util.ResourceClient, targetNamespace string, triggerFunc func(pkgruntime.Object)) PlacementPlugin {
	return &namespacedPlacementPlugin{
		resourcePlugin:  NewResourcePlacementPlugin(resourceClient, targetNamespace, triggerFunc, false),
		namespacePlugin: newResourcePlacementPluginWithOk(namespaceClient, targetNamespace, triggerFunc, false),
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
	if !ok {
		// Use the resource placement if no namespace placement is
		// available.
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
