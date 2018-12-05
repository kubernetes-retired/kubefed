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

package schedulingmanager

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	corev1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	corev1alpha1client "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned/typed/core/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/schedulingpreference"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/schedulingtypes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
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

	scheduler map[string]schedulingtypes.Scheduler

	config   *util.ControllerConfig
	stopChan <-chan struct{}

	runningPlugins sets.String
}

func StartSchedulerController(config *util.ControllerConfig, stopChan <-chan struct{}) {

	userAgent := "SchedulerController"
	kubeConfig := config.KubeConfig
	restclient.AddUserAgent(kubeConfig, userAgent)
	client := fedclientset.NewForConfigOrDie(kubeConfig).CoreV1alpha1()

	controller := newController(config, client, stopChan)

	glog.Infof("Starting scheduler controller")
	controller.Run(stopChan)
}

func newController(config *util.ControllerConfig, client corev1alpha1client.CoreV1alpha1Interface, stopChan <-chan struct{}) *SchedulerController {
	c := &SchedulerController{
		scheduler:      make(map[string]schedulingtypes.Scheduler),
		config:         config,
		stopChan:       stopChan,
		runningPlugins: sets.String{},
	}

	fedNamespace := config.FederationNamespace
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

	return c
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

	schedulingType := schedulingtypes.GetSchedulingType(qualifiedName.Name)
	if schedulingType == nil {
		// No scheduler supported for this resource
		return util.StatusAllOK
	}
	if c.runningPlugins.Has(qualifiedName.Name) {
		// Scheduler and plugin are already running
		return util.StatusAllOK
	}

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

	// set name and group for the type config target
	corev1a1.SetFederatedTypeConfigDefaults(typeConfig)

	// TODO(marun) Replace with validation webhook
	err = typeconfig.CheckTypeConfigName(typeConfig)
	if err != nil {
		runtime.HandleError(err)
		return util.StatusError
	}

	// Scheduling preference controller is started on demand
	schedulingKind := schedulingType.Kind
	scheduler, ok := c.scheduler[schedulingKind]
	if !ok {
		var err error
		scheduler, err = schedulingpreference.StartSchedulingPreferenceController(c.config, *schedulingType, c.stopChan)
		if err != nil {
			runtime.HandleError(fmt.Errorf("Error starting schedulingpreference controller for %q : %v", schedulingKind, err))
			return util.StatusError
		}
		c.scheduler[schedulingKind] = scheduler
	}

	templateKind := typeConfig.GetTemplate().Kind
	glog.Infof("Start plugin with kind %s for scheduling type %s", templateKind, schedulingKind)
	err = scheduler.StartPlugin(typeConfig, c.stopChan)
	if err != nil {
		runtime.HandleError(fmt.Errorf("Error starting plugin for %q : %v", templateKind, err))
		return util.StatusError
	}
	c.runningPlugins.Insert(qualifiedName.Name)

	return util.StatusAllOK
}
