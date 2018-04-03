/*
Copyright 2017 The Kubernetes Authors.

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

package federatedtypes

import (
	fedclientset "github.com/marun/federation-v2/pkg/client/clientset_generated/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	kubeclientset "k8s.io/client-go/kubernetes"
)

// FederatedTypeAdapter provides a common interface for interacting with
// the component resources of a federated type and its non-federated
// target resource in member clusters.
type FederatedTypeAdapter interface {
	FedClient() fedclientset.Interface
	Template() FedApiAdapter
	Placement() PlacementAdapter
	Override() OverrideAdapter
	Target() TargetAdapter
	ObjectForCluster(template, override pkgruntime.Object, clusterName string) pkgruntime.Object
}

type MetaAdapter interface {
	Kind() string
	ObjectMeta(pkgruntime.Object) *metav1.ObjectMeta
	ObjectType() pkgruntime.Object
}

type FedApiAdapter interface {
	MetaAdapter

	// Client methods for accessing the type in the federation api
	Create(obj pkgruntime.Object) (pkgruntime.Object, error)
	Delete(qualifiedName QualifiedName, options *metav1.DeleteOptions) error
	Get(qualifiedName QualifiedName) (pkgruntime.Object, error)
	List(namespace string, options metav1.ListOptions) (pkgruntime.Object, error)
	Update(obj pkgruntime.Object) (pkgruntime.Object, error)
	Watch(namespace string, options metav1.ListOptions) (watch.Interface, error)
}

type PlacementAdapter interface {
	FedApiAdapter

	ClusterNames(obj pkgruntime.Object) []string
	SetClusterNames(obj pkgruntime.Object, clusterNames []string)
}

type OverrideAdapter interface {
	FedApiAdapter
}

type TargetAdapter interface {
	MetaAdapter

	// Client methods for accessing the type in member clusters
	Create(client kubeclientset.Interface, obj pkgruntime.Object) (pkgruntime.Object, error)
	Delete(client kubeclientset.Interface, qualifiedName QualifiedName, options *metav1.DeleteOptions) error
	Get(client kubeclientset.Interface, qualifiedName QualifiedName) (pkgruntime.Object, error)
	List(client kubeclientset.Interface, namespace string, options metav1.ListOptions) (pkgruntime.Object, error)
	Update(client kubeclientset.Interface, obj pkgruntime.Object) (pkgruntime.Object, error)
	Watch(client kubeclientset.Interface, namespace string, options metav1.ListOptions) (watch.Interface, error)
}
