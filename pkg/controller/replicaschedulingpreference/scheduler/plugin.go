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

package scheduler

import (
	"fmt"

	"github.com/golang/glog"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset_generated/clientset"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	crclientset "k8s.io/cluster-registry/pkg/client/clientset/versioned"
)

type Plugin struct {
	targetInformer util.FederatedInformer

	templateStore cache.Store
	// Informer for the templates of the federated type
	templateController cache.Controller

	// Store for the override directives of the federated type
	overrideStore cache.Store
	// Informer controller for override directives of the federated type
	overrideController cache.Controller

	// Store for the placements of the federated type
	placementStore cache.Store
	// Informer controller for placements of the federated type
	placementController cache.Controller

	adapter SchedulerAdapter
}

func NewPlugin(adapter SchedulerAdapter, apiResource *metav1.APIResource, fedClient fedclientset.Interface, kubeClient kubeclientset.Interface, crClient crclientset.Interface, federationEventHandler, clusterEventHandler func(pkgruntime.Object), handlers *util.ClusterLifecycleHandlerFuncs) *Plugin {
	p := &Plugin{
		targetInformer: util.NewFederatedInformer(
			fedClient,
			kubeClient,
			crClient,
			apiResource,
			clusterEventHandler,
			handlers,
		),
		adapter: adapter,
	}

	p.templateStore, p.templateController = cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (pkgruntime.Object, error) {
				return adapter.TemplateList(metav1.NamespaceAll, options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return adapter.TemplateWatch(metav1.NamespaceAll, options)
			},
		},
		adapter.TemplateObject(),
		util.NoResyncPeriod,
		util.NewTriggerOnAllChanges(federationEventHandler),
	)

	p.overrideStore, p.overrideController = cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (pkgruntime.Object, error) {
				return adapter.OverrideList(metav1.NamespaceAll, options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return adapter.OverrideWatch(metav1.NamespaceAll, options)
			},
		},
		adapter.OverrideObject(),
		util.NoResyncPeriod,
		util.NewTriggerOnAllChanges(federationEventHandler),
	)

	p.placementStore, p.placementController = cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (pkgruntime.Object, error) {
				return adapter.PlacementList(metav1.NamespaceAll, options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return adapter.PlacementWatch(metav1.NamespaceAll, options)
			},
		},
		adapter.PlacementObject(),
		util.NoResyncPeriod,
		util.NewTriggerOnAllChanges(federationEventHandler),
	)

	return p
}

func (p *Plugin) Start(stopChan <-chan struct{}) {
	p.targetInformer.Start()
	go p.templateController.Run(stopChan)
	go p.overrideController.Run(stopChan)
	go p.placementController.Run(stopChan)
}

func (p *Plugin) Stop() {
	p.targetInformer.Stop()
}

func (p *Plugin) HasSynced() bool {
	if !p.targetInformer.ClustersSynced() {
		glog.V(2).Infof("Cluster list not synced")
		return false
	}
	clusters, err := p.targetInformer.GetReadyClusters()
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to get ready clusters: %v", err))
		return false
	}

	return p.targetInformer.GetTargetStore().ClustersSynced(clusters)
}

func (p *Plugin) TemplateExists(key string) bool {
	_, exist, err := p.templateStore.GetByKey(key)
	if err != nil {
		glog.Errorf("Failed to query store while reconciling RSP controller for key %q: %v", key, err)
		wrappedErr := fmt.Errorf("Failed to query store while reconciling RSP controller for key %q: %v", key, err)
		runtime.HandleError(wrappedErr)
		return false
	}
	return exist
}
