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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type ResourcePlacementPlugin struct {
	// Store for the placement directives of the federated type
	store cache.Store
	// Informer controller for placement directives of the federated type
	controller cache.Controller
}

func NewResourcePlacementPlugin(config *rest.Config, resource schema.GroupVersionResource, triggerFunc func(*unstructured.Unstructured)) (PlacementPlugin, error) {
	store, controller, err := NewUnstructuredInformer(config, resource, triggerFunc)
	if err != nil {
		return nil, err
	}

	return &ResourcePlacementPlugin{
		store:      store,
		controller: controller,
	}, nil
}

func (p *ResourcePlacementPlugin) Run(stopCh <-chan struct{}) {
	p.controller.Run(stopCh)
}

func (p *ResourcePlacementPlugin) HasSynced() bool {
	return p.controller.HasSynced()
}

func (p *ResourcePlacementPlugin) ComputePlacement(key string, clusterNames []string) (selectedClusters, unselectedClusters []string, err error) {
	cachedObj, _, err := p.store.GetByKey(key)
	if err != nil {
		return nil, nil, err
	}
	if cachedObj == nil {
		return []string{}, clusterNames, nil
	}
	unstructuredObj := cachedObj.(*unstructured.Unstructured)

	// TODO (font): NestedStringSlice returns false if the clusternames field
	// value is not found, which can happen when the clusternames field is
	// empty i.e. when a user does not want to propagate the resource anywhere.
	// Therefore, ignore the ok return value for now as we'll expect false
	// returned only in the event the clusternames field is empty, which is a
	// valid use-case. Ideally, we should not avoid a false return and expand
	// or re-write NestedStringSlice to check for the empty case as well as to
	// make sure the unstructured object in-fact has a proper "spec" and
	// "clusternames" field to avoid any accidental typos in the creation of a
	// propagation resource.
	selectedNames, _ := unstructured.NestedStringSlice(unstructuredObj.Object, "spec", "clusternames")
	clusterSet := sets.NewString(clusterNames...)
	selectedSet := sets.NewString(selectedNames...)
	return clusterSet.Intersection(selectedSet).List(), clusterSet.Difference(selectedSet).List(), nil
}
