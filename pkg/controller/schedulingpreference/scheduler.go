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

package schedulingpreference

import (
	"fmt"

	"github.com/golang/glog"
	corev1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	corev1alpha1client "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned/typed/core/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/schedulingtypes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type SchedulerController struct {
	// Store for the FederatedTypeConfig objects
	store cache.Store
	// Informer for the FederatedTypeConfig objects
	controller cache.Controller

	worker util.ReconcileWorker

	scheduler *map[string]schedulingtypes.Scheduler
	stopChan  <-chan struct{}
}

func StartSchedulerController(config *restclient.Config, fedNamespace, clusterNamespace, targetNamespace string, stopChan <-chan struct{}, minimizeLatency bool) error {

	s := make(map[string]schedulingtypes.Scheduler)

	for kind, schedulingType := range schedulingtypes.SchedulingTypes() {
		scheduler, err := StartSchedulingPreferenceController(kind, schedulingType.SchedulerFactory, config, fedNamespace, clusterNamespace, targetNamespace, stopChan, true)
		if err != nil {
			glog.Fatalf("Error starting schedulingpreference controller for %q : %v", kind, err)
		}
		s[kind] = scheduler
	}

	userAgent := "SchedulerController"
	restclient.AddUserAgent(config, userAgent)
	client := fedclientset.NewForConfigOrDie(config).CoreV1alpha1()

	controller, err := newController(client, config, fedNamespace, &s)
	if err != nil {
		return err
	}
	glog.Infof("Starting scheduler controller")

	controller.stopChan = stopChan
	controller.Run(stopChan)

	return nil
}

func newController(client corev1alpha1client.CoreV1alpha1Interface, config *restclient.Config, fedNamespace string, scheduler *map[string]schedulingtypes.Scheduler) (*SchedulerController, error) {
	c := &SchedulerController{
		scheduler: scheduler,
	}

	c.worker = util.NewReconcileWorker(c.reconcile, util.WorkerTiming{})

	c.store, c.controller = cache.NewInformer(
		&cache.ListWatch{
			// Only watch the federation namespace to ensure
			// restrictive authz can be applied to a namespaced
			// control plane.
			ListFunc: func(options metav1.ListOptions) (pkgruntime.Object, error) {
				return client.FederatedTypeConfigs(fedNamespace).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return client.FederatedTypeConfigs(fedNamespace).Watch(options)
			},
		},
		&corev1a1.FederatedTypeConfig{},
		util.NoResyncPeriod,
		util.NewTriggerOnAllChanges(c.worker.EnqueueObject),
	)

	return c, nil
}

// Run runs the Controller.
func (c *SchedulerController) Run(stopChan <-chan struct{}) {
	go c.controller.Run(stopChan)

	// wait for the caches to synchronize before starting the worker
	if !cache.WaitForCacheSync(stopChan, c.controller.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for cache to sync"))
		return
	}

	c.worker.Run(stopChan)
}

func (c *SchedulerController) reconcile(qualifiedName util.QualifiedName) util.ReconciliationStatus {
	key := qualifiedName.String()

	glog.Infof("Running reconcile FederatedTypeConfig for %q", key)

	cachedObj, exist, err := c.store.GetByKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to query FederatedTypeConfig store for %q: %v", key, err))
		return util.StatusError
	}
	if !exist {
		return util.StatusAllOK
	}
	typeConfig := cachedObj.(*corev1a1.FederatedTypeConfig)

	deleted := typeConfig.DeletionTimestamp != nil
	if deleted {
		// Do not support deletion yet
		return util.StatusAllOK
	}

	apiResource := typeConfig.GetTarget()
	kind := typeConfig.GetTemplate().Kind

	for name, _ := range schedulingtypes.SchedulingTypes() {
		glog.Infof("Trying to register plugins for scheduler type %s", name)
		(*c.scheduler)[name].RegisterPlugins(kind, apiResource, c.stopChan)
	}
	return util.StatusAllOK
}
