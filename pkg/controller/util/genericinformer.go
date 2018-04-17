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

package util

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
)

func resourceInterface(client dynamic.Interface, resource string) dynamic.ResourceInterface {
	// APIResource.Kind is not used by the dynamic client, so
	// leave it empty. We want to list this resource in all
	// namespaces if it's namespace scoped, so leaving
	// APIResource.Namespaced as false is all right.
	apiResource := metav1.APIResource{Name: resource}
	return client.ParameterCodec(dynamic.VersionedParameterEncoderWithV1Fallback).
		Resource(&apiResource, metav1.NamespaceAll)
}

func NewGenericInformer(client dynamic.Interface, resource string, triggerFunc func(interface{})) (cache.Store, cache.Controller) {
	return cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return resourceInterface(client, resource).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return resourceInterface(client, resource).Watch(options)
			},
		},
		nil, // Skip checks for expected type since the type will depend on resource
		NoResyncPeriod,
		&cache.ResourceEventHandlerFuncs{
			DeleteFunc: func(obj interface{}) {
				triggerFunc(obj)
			},
			AddFunc: func(obj interface{}) {
				triggerFunc(obj)
			},
			UpdateFunc: func(old, cur interface{}) {
				if reflect.DeepEqual(old, cur) {
					return
				}
				triggerFunc(cur)
			},
		},
	)
}
