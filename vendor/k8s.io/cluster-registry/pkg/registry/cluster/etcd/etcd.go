/*
Copyright 2016 The Kubernetes Authors.

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

// Package etcd contains an implementation of a store for Cluster objects.
package etcd

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
	"k8s.io/cluster-registry/pkg/registry/cluster"
)

type REST struct {
	*genericregistry.Store
}

// NewREST returns a RESTStorage object that will work against Cluster objects.
func NewREST(optsGetter generic.RESTOptionsGetter, scheme *runtime.Scheme) (*REST, error) {
	store := &genericregistry.Store{
		NewFunc:                  func() runtime.Object { return &clusterregistry.Cluster{} },
		NewListFunc:              func() runtime.Object { return &clusterregistry.ClusterList{} },
		PredicateFunc:            cluster.MatchCluster,
		DefaultQualifiedResource: clusterregistry.Resource("clusters"),

		CreateStrategy: cluster.Strategy,
		UpdateStrategy: cluster.Strategy,
		DeleteStrategy: cluster.Strategy,
	}
	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: cluster.GetAttrs}
	if err := store.CompleteWithOptions(options); err != nil {
		return nil, err
	}

	return &REST{store}, nil
}
