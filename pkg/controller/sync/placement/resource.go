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
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"
)

type ResourcePlacementPlugin struct {
	// Store for the placement directives of the federated type
	store cache.Store
	// Informer controller for placement directives of the federated type
	controller cache.Controller
}

func NewResourcePlacementPlugin(client util.ResourceClient, targetNamespace string, triggerFunc func(pkgruntime.Object)) PlacementPlugin {
	return newResourcePlacementPluginWithOk(client, targetNamespace, triggerFunc)
}

func newResourcePlacementPluginWithOk(client util.ResourceClient, targetNamespace string, triggerFunc func(pkgruntime.Object)) *ResourcePlacementPlugin {
	store, controller := util.NewResourceInformer(client, targetNamespace, triggerFunc)
	return &ResourcePlacementPlugin{
		store:      store,
		controller: controller,
	}
}

func (p *ResourcePlacementPlugin) Run(stopCh <-chan struct{}) {
	p.controller.Run(stopCh)
}

func (p *ResourcePlacementPlugin) HasSynced() bool {
	return p.controller.HasSynced()
}

func (p *ResourcePlacementPlugin) ComputePlacement(qualifiedName util.QualifiedName, clusterNames []string) (selectedClusters, unselectedClusters []string, err error) {
	selectedClusters, unselectedClusters, _, err = p.computePlacementWithOk(qualifiedName, clusterNames)
	return selectedClusters, unselectedClusters, err
}

func (p *ResourcePlacementPlugin) computePlacementWithOk(qualifiedName util.QualifiedName, clusterNames []string) (selectedClusters, unselectedClusters []string, ok bool, err error) {
	key := qualifiedName.String()
	cachedObj, _, err := p.store.GetByKey(key)
	if err != nil {
		return nil, nil, false, err
	}
	if cachedObj == nil {
		return []string{}, clusterNames, false, nil
	}
	unstructuredObj := cachedObj.(*unstructured.Unstructured)

	selectedNames, err := util.GetClusterNames(unstructuredObj)
	if err != nil {
		return nil, nil, false, err
	}
	clusterSet := sets.NewString(clusterNames...)
	selectedSet := sets.NewString(selectedNames...)
	return clusterSet.Intersection(selectedSet).List(), clusterSet.Difference(selectedSet).List(), true, nil
}
