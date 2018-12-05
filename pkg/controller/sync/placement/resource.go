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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"
)

type ResourcePlacementPlugin struct {
	// Store for the placement directives of the federated type
	store cache.Store
	// Informer controller for placement directives of the federated type
	controller cache.Controller

	// Whether to default to all clusters if a placement resource is
	// not found for a given qualified name.
	defaultAll bool
}

func NewResourcePlacementPlugin(client util.ResourceClient, targetNamespace string, triggerFunc func(pkgruntime.Object), defaultAll bool) PlacementPlugin {
	return newResourcePlacementPluginWithOk(client, targetNamespace, triggerFunc, defaultAll)
}

func newResourcePlacementPluginWithOk(client util.ResourceClient, targetNamespace string, triggerFunc func(pkgruntime.Object), defaultAll bool) *ResourcePlacementPlugin {
	store, controller := util.NewResourceInformer(client, targetNamespace, triggerFunc)
	return &ResourcePlacementPlugin{
		store:      store,
		controller: controller,
		defaultAll: defaultAll,
	}
}

func (p *ResourcePlacementPlugin) Run(stopCh <-chan struct{}) {
	p.controller.Run(stopCh)
}

func (p *ResourcePlacementPlugin) HasSynced() bool {
	return p.controller.HasSynced()
}

func (p *ResourcePlacementPlugin) ComputePlacement(qualifiedName util.QualifiedName, clusters []*fedv1a1.FederatedCluster) (selectedClusters, unselectedClusters []string, err error) {
	selectedClusters, unselectedClusters, _, err = p.computePlacementWithOk(qualifiedName, clusters)
	return selectedClusters, unselectedClusters, err
}

func (p *ResourcePlacementPlugin) computePlacementWithOk(qualifiedName util.QualifiedName, clusters []*fedv1a1.FederatedCluster) (selectedClusters, unselectedClusters []string, ok bool, err error) {
	key := qualifiedName.String()
	cachedObj, _, err := p.store.GetByKey(key)
	if err != nil {
		return nil, nil, false, err
	}
	clusterNames := getClusterNames(clusters)
	if cachedObj == nil {
		if p.defaultAll {
			return clusterNames, []string{}, false, nil
		}
		return []string{}, clusterNames, false, nil
	}
	unstructuredObj := cachedObj.(*unstructured.Unstructured)

	selectedNames, err := selectedClusterNames(unstructuredObj, clusters)
	if err != nil {
		return nil, nil, false, err
	}

	clusterSet := sets.NewString(clusterNames...)
	selectedSet := sets.NewString(selectedNames...)
	return clusterSet.Intersection(selectedSet).List(), clusterSet.Difference(selectedSet).List(), true, nil
}

func getClusterNames(clusters []*fedv1a1.FederatedCluster) []string {
	clusterNames := []string{}
	for _, cluster := range clusters {
		clusterNames = append(clusterNames, cluster.Name)
	}
	return clusterNames
}

func selectedClusterNames(rawPlacement *unstructured.Unstructured, clusters []*fedv1a1.FederatedCluster) ([]string, error) {
	directive, err := util.GetPlacementDirective(rawPlacement)
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
