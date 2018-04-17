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

package placement

import (
	"github.com/marun/federation-v2/pkg/controller/util"
	"github.com/marun/federation-v2/pkg/federatedtypes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

type NamespacePlacementPlugin struct {
	adapter federatedtypes.PlacementAdapter
	// Store for the placement directives of the federated type
	store cache.Store
	// Informer controller for placement directives of the federated type
	controller cache.Controller
}

func NewNamespacePlacementPlugin(adapter federatedtypes.PlacementAdapter, triggerFunc func(pkgruntime.Object)) PlacementPlugin {
	store, controller := cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (pkgruntime.Object, error) {
				return adapter.List(metav1.NamespaceAll, options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return adapter.Watch(metav1.NamespaceAll, options)
			},
		},
		adapter.ObjectType(),
		util.NoResyncPeriod,
		util.NewTriggerOnAllChanges(triggerFunc),
	)

	return &NamespacePlacementPlugin{
		adapter:    adapter,
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
	placement := cachedObj.(pkgruntime.Object)

	clusterSet := sets.NewString(clusterNames...)
	selectedClusterSet := sets.NewString(p.adapter.ClusterNames(placement)...)
	return clusterSet.Intersection(selectedClusterSet).List(), clusterSet.Difference(selectedClusterSet).List(), nil
}
