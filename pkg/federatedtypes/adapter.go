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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	kubeclientset "k8s.io/client-go/kubernetes"
)

// FederatedTypeAdapter provides a common interface for interacting
// with federated types in the federation api and its non-federated
// target in memer clusters.
type FederatedTypeAdapter interface {
	// Methods applying to federated types

	FedKind() string
	FedObjectMeta(pkgruntime.Object) *metav1.ObjectMeta
	FedObjectType() pkgruntime.Object
	ObjectForCluster(obj pkgruntime.Object, clusterName string) pkgruntime.Object

	FedCreate(obj pkgruntime.Object) (pkgruntime.Object, error)
	FedDelete(qualifiedName QualifiedName, options *metav1.DeleteOptions) error
	FedGet(qualifiedName QualifiedName) (pkgruntime.Object, error)
	FedList(namespace string, options metav1.ListOptions) (pkgruntime.Object, error)
	FedUpdate(obj pkgruntime.Object) (pkgruntime.Object, error)
	FedWatch(namespace string, options metav1.ListOptions) (watch.Interface, error)

	// Methods applying to non-federated types

	Kind() string
	ObjectMeta(pkgruntime.Object) *metav1.ObjectMeta
	ObjectType() pkgruntime.Object
	Equivalent(obj1, obj2 pkgruntime.Object) bool

	Create(client kubeclientset.Interface, obj pkgruntime.Object) (pkgruntime.Object, error)
	Delete(client kubeclientset.Interface, qualifiedName QualifiedName, options *metav1.DeleteOptions) error
	Get(client kubeclientset.Interface, qualifiedName QualifiedName) (pkgruntime.Object, error)
	List(client kubeclientset.Interface, namespace string, options metav1.ListOptions) (pkgruntime.Object, error)
	Update(client kubeclientset.Interface, obj pkgruntime.Object) (pkgruntime.Object, error)
	Watch(client kubeclientset.Interface, namespace string, options metav1.ListOptions) (watch.Interface, error)
}
