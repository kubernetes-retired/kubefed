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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"
)

type NamespacePlacementPlugin struct {
	// Store for the placement directives of the federated type
	store cache.Store
	// Informer controller for placement directives of the federated type
	controller cache.Controller
}

func NewNamespacePlacementPlugin(client util.ResourceClient, triggerFunc func(pkgruntime.Object)) PlacementPlugin {
	store, controller := util.NewResourceInformer(client, metav1.NamespaceAll, triggerFunc)
	return &NamespacePlacementPlugin{
		store:      store,
		controller: controller,
	}
}

func (p *NamespacePlacementPlugin) Run(stopCh <-chan struct{}) {
	p.controller.Run(stopCh)
}

func (p *NamespacePlacementPlugin) HasSynced() bool {
	return p.controller.HasSynced()
}

func (p *NamespacePlacementPlugin) ComputePlacement(key string, clusterNames []string) (selectedClusters, unselectedClusters []string, err error) {
	cachedObj, _, err := p.store.GetByKey(key)
	if err != nil {
		return nil, nil, err
	}
	if cachedObj == nil {
		// TODO(marun) Compute placement from the placement decisions of contained resources
		return clusterNames, []string{}, nil
	}
	unstructuredObj := cachedObj.(*unstructured.Unstructured)

	selectedNames := util.GetClusterNames(unstructuredObj)
	clusterSet := sets.NewString(clusterNames...)
	selectedSet := sets.NewString(selectedNames...)
	return clusterSet.Intersection(selectedSet).List(), clusterSet.Difference(selectedSet).List(), nil
}
