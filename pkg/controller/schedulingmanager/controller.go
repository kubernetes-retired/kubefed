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
	"sync"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	corev1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/schedulingpreference"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/schedulingtypes"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type SchedulerController struct {
	sync.RWMutex
	// Store for the FederatedTypeConfig objects
	store cache.Store
	// Informer for the FederatedTypeConfig objects
	controller cache.Controller

	worker util.ReconcileWorker

	scheduler map[string]schedulingtypes.Scheduler

	config *util.ControllerConfig

	runningPlugins sets.String
	// mapping qualifiedname to template kind for managing plugins in scheduler
	federatedKindMap map[string]string
}

func StartSchedulerController(config *util.ControllerConfig, stopChan <-chan struct{}) (*SchedulerController, error) {

	userAgent := "SchedulerController"
	kubeConfig := config.KubeConfig
	restclient.AddUserAgent(kubeConfig, userAgent)

	controller, err := newController(config)
	if err != nil {
		return nil, err
	}

	glog.Infof("Starting scheduler controller")
	controller.Run(stopChan)
	return controller, nil
}

func newController(config *util.ControllerConfig) (*SchedulerController, error) {
	c := &SchedulerController{
		config:           config,
		scheduler:        make(map[string]schedulingtypes.Scheduler),
		runningPlugins:   make(sets.String),
		federatedKindMap: make(map[string]string),
	}

	c.worker = util.NewReconcileWorker(c.reconcile, util.WorkerTiming{})

	var err error
	c.store, c.controller, err = util.NewGenericInformer(
		config.KubeConfig,
		config.FederationNamespace,
		&corev1a1.FederatedTypeConfig{},
		util.NoResyncPeriod,
		c.worker.EnqueueObject,
	)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// Run runs the Controller.
func (c *SchedulerController) Run(stopChan <-chan struct{}) {
	go c.controller.Run(stopChan)

	// wait for the caches to synchronize before starting the worker
	if !cache.WaitForCacheSync(stopChan, c.controller.HasSynced) {
		runtime.HandleError(errors.New("Timed out waiting for cache to sync"))
		return
	}

	c.worker.Run(stopChan)
}

func (c *SchedulerController) reconcile(qualifiedName util.QualifiedName) util.ReconciliationStatus {
	key := qualifiedName.String()

	glog.V(3).Infof("Running reconcile FederatedTypeConfig for %q", key)

	schedulingType := schedulingtypes.GetSchedulingType(qualifiedName.Name)
	if schedulingType == nil {
		// No scheduler supported for this resource
		return util.StatusAllOK
	}

	cachedObj, exist, err := c.store.GetByKey(key)
	if err != nil {
		runtime.HandleError(errors.Wrapf(err, "Failed to query FederatedTypeConfig store for %q", key))
		return util.StatusError
	}

	if !exist {
		c.stopScheduler(schedulingType.Kind, qualifiedName)
		return util.StatusAllOK
	}

	typeConfig := cachedObj.(*corev1a1.FederatedTypeConfig)
	if typeConfig.Spec.PropagationEnabled == false || typeConfig.DeletionTimestamp != nil {
		c.stopScheduler(schedulingType.Kind, qualifiedName)
		return util.StatusAllOK
	}

	c.Lock()
	defer c.Unlock()
	if c.runningPlugins.Has(qualifiedName.Name) {
		// Scheduler and plugin are already running
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

		scheduler, err = schedulingpreference.StartSchedulingPreferenceController(c.config, *schedulingType)
		if err != nil {
			runtime.HandleError(errors.Wrapf(err, "Error starting schedulingpreference controller for %q", schedulingKind))
			return util.StatusError
		}
		c.scheduler[schedulingKind] = scheduler
	}

	federatedKind := typeConfig.GetFederatedType().Kind
	glog.Infof("Starting plugin kind %s for scheduling type %s", federatedKind, schedulingKind)

	err = scheduler.StartPlugin(typeConfig)
	if err != nil {
		runtime.HandleError(errors.Wrapf(err, "Error starting plugin for %q", federatedKind))
		return util.StatusError
	}
	c.runningPlugins.Insert(qualifiedName.Name)
	c.federatedKindMap[qualifiedName.Name] = federatedKind

	return util.StatusAllOK
}

func (c *SchedulerController) stopScheduler(schedulingKind string, qualifiedName util.QualifiedName) {
	c.Lock()
	defer c.Unlock()
	scheduler, ok := c.scheduler[schedulingKind]
	if !ok {
		return
	}

	if !c.runningPlugins.Has(qualifiedName.Name) {
		return
	}

	kind, ok := c.federatedKindMap[qualifiedName.Name]
	if ok {
		glog.Infof("Stopping plugin for %q with kind %q", qualifiedName.Name, kind)
		scheduler.StopPlugin(kind)
		delete(c.federatedKindMap, qualifiedName.Name)
	}
	c.runningPlugins.Delete(qualifiedName.Name)

	// if all resources registered to same scheduler are deleted, the scheduler should be stopped
	resources := schedulingtypes.GetSchedulingKinds(qualifiedName.Name)
	result := c.runningPlugins.Intersection(resources)
	if result.Len() == 0 {
		glog.Infof("Stopping scheduler schedulingpreference controller for %q", schedulingKind)
		scheduler.Stop()

		delete(c.scheduler, schedulingKind)
	}
}

func (c *SchedulerController) HasSchedulerPlugin(name string) bool {
	c.RLock()
	defer c.RUnlock()
	return c.runningPlugins.Has(name)
}

func (c *SchedulerController) HasScheduler(name string) bool {
	c.RLock()
	defer c.RUnlock()
	_, ok := c.scheduler[name]
	if !ok {
		return false
	}
	return true
}
