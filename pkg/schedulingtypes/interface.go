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

package schedulingtypes

import (
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	. "github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	kubeclientset "k8s.io/client-go/kubernetes"
	crclientset "k8s.io/cluster-registry/pkg/client/clientset/versioned"
)

type Scheduler interface {
	Kind() string
	ObjectType() pkgruntime.Object
	FedList(namespace string, options metav1.ListOptions) (pkgruntime.Object, error)
	FedWatch(namespace string, options metav1.ListOptions) (watch.Interface, error)

	Start(stopChan <-chan struct{})
	HasSynced() bool
	Stop()
	Reconcile(obj pkgruntime.Object, qualifiedName QualifiedName) ReconciliationStatus

	RegisterPlugins(kind string, apiResource metav1.APIResource, stopChan <-chan struct{})
}

type SchedulerFactory func(fedClient fedclientset.Interface, kubeClient kubeclientset.Interface, crClient crclientset.Interface, fedNamespace, clusterNamespace, targetNamespace string, federationEventHandler, clusterEventHandler func(pkgruntime.Object), handlers *ClusterLifecycleHandlerFuncs) Scheduler
