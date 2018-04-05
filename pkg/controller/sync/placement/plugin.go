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
	"github.com/marun/federation-v2/pkg/controller/util"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type PlacementPlugin interface {
	Run(stopCh <-chan struct{})
	HasSynced() bool
	ComputePlacement(key string, clusterNames []string) (selectedClusters, unselectedClusters []string, err error)
}

func NewUnstructuredInformer(config *rest.Config, resource schema.GroupVersionResource, triggerFunc func(*unstructured.Unstructured)) (cache.Store, cache.Controller, error) {
	// Ensure the correct api path is used - the default is /api
	config.APIPath = "/apis"
	// The dynamic client requires the group version to be set
	groupVersion := resource.GroupVersion()
	config.GroupVersion = &groupVersion

	client, err := dynamic.NewClient(config)
	if err != nil {
		return nil, nil, err
	}

	store, controller := util.NewGenericInformer(client, resource.Resource, func(interfaceObj interface{}) {
		var obj *unstructured.Unstructured
		if interfaceObj != nil {
			obj = interfaceObj.(*unstructured.Unstructured)
		}
		triggerFunc(obj)
	})
	return store, controller, nil
}
