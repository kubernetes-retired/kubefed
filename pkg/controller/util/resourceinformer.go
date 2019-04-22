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

package util

import (
	"github.com/pkg/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// NewManagedResourceInformer returns an unfiltered informer.
func NewResourceInformer(client ResourceClient, namespace string, triggerFunc func(pkgruntime.Object)) (cache.Store, cache.Controller) {
	return newResourceInformer(client, namespace, triggerFunc, "")
}

// NewManagedResourceInformer returns an informer limited to resources
// managed by federation as indicated by labeling.
func NewManagedResourceInformer(client ResourceClient, namespace string, triggerFunc func(pkgruntime.Object)) (cache.Store, cache.Controller) {
	labelSelector := labels.Set(map[string]string{ManagedByFederationLabelKey: ManagedByFederationLabelValue}).AsSelector().String()
	return newResourceInformer(client, namespace, triggerFunc, labelSelector)
}

func newResourceInformer(client ResourceClient, namespace string, triggerFunc func(pkgruntime.Object), labelSelector string) (cache.Store, cache.Controller) {
	return cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (pkgruntime.Object, error) {
				options.LabelSelector = labelSelector
				return client.Resources(namespace).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				options.LabelSelector = labelSelector
				return client.Resources(namespace).Watch(options)
			},
		},
		nil, // Skip checks for expected type since the type will depend on the client
		NoResyncPeriod,
		NewTriggerOnAllChanges(triggerFunc),
	)
}

func ObjFromCache(store cache.Store, kind, key string) (*unstructured.Unstructured, error) {
	obj, err := rawObjFromCache(store, kind, key)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, nil
	}
	return obj.(*unstructured.Unstructured), nil
}

func rawObjFromCache(store cache.Store, kind, key string) (pkgruntime.Object, error) {
	cachedObj, exist, err := store.GetByKey(key)
	if err != nil {
		wrappedErr := errors.Wrapf(err, "Failed to query %s store for %q", kind, key)
		runtime.HandleError(wrappedErr)
		return nil, err
	}
	if !exist {
		return nil, nil
	}
	return cachedObj.(pkgruntime.Object).DeepCopyObject(), nil
}
