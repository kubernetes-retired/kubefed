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
	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/util/runtime"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"

	corev1b1 "sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	"sigs.k8s.io/kubefed/pkg/controller/schedulingpreference"
	"sigs.k8s.io/kubefed/pkg/controller/util"
	"sigs.k8s.io/kubefed/pkg/schedulingtypes"
)

type SchedulingManager struct {
	// Store for the FederatedTypeConfig objects
	store cache.Store
	// Informer for the FederatedTypeConfig objects
	controller cache.Controller

	worker util.ReconcileWorker

	//schedulers map[string]*SchedulerWrapper
	schedulers *util.SafeMap

	config *util.ControllerConfig
}

type SchedulerWrapper struct {
	// To signal shutdown of scheduler and any associated routine.
	stopChan chan struct{}
	// Mapping qualifiedname to federated kind for managing plugins in scheduler.
	// This is needed because typeconfig could be of any name and we run plugins
	// by federated kinds (eg FederatedDeployment). This also avoids running multiple
	// plugins in case multiple typeconfigs are created for same federated kind.
	pluginMap *util.SafeMap
	// Actual scheduler.
	schedulingtypes.Scheduler
}

func (s *SchedulerWrapper) HasPlugin(typeConfigName string) bool {
	_, ok := s.pluginMap.Get(typeConfigName)
	return ok
}

func StartSchedulingManager(config *util.ControllerConfig, stopChan <-chan struct{}) (*SchedulingManager, error) {
	manager, err := newSchedulingManager(config)
	if err != nil {
		return nil, err
	}

	klog.Infof("Starting scheduling manager")
	manager.Run(stopChan)
	return manager, nil
}

func newSchedulerWrapper(schedulerInterface schedulingtypes.Scheduler, stopChan chan struct{}) *SchedulerWrapper {
	return &SchedulerWrapper{
		stopChan:  stopChan,
		pluginMap: util.NewSafeMap(),
		Scheduler: schedulerInterface,
	}
}

func newSchedulingManager(config *util.ControllerConfig) (*SchedulingManager, error) {
	userAgent := "SchedulingManager"
	kubeConfig := restclient.CopyConfig(config.KubeConfig)
	restclient.AddUserAgent(kubeConfig, userAgent)

	c := &SchedulingManager{
		config:     config,
		schedulers: util.NewSafeMap(),
	}

	c.worker = util.NewReconcileWorker(c.reconcile, util.WorkerTiming{})

	var err error
	c.store, c.controller, err = util.NewGenericInformer(
		kubeConfig,
		config.KubeFedNamespace,
		&corev1b1.FederatedTypeConfig{},
		util.NoResyncPeriod,
		c.worker.EnqueueObject,
	)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *SchedulingManager) GetScheduler(schedulingKind string) *SchedulerWrapper {
	scheduler, ok := c.schedulers.Get(schedulingKind)
	if !ok {
		return nil
	}
	return scheduler.(*SchedulerWrapper)
}

// Run runs the Controller.
func (c *SchedulingManager) Run(stopChan <-chan struct{}) {
	go c.controller.Run(stopChan)

	// wait for the caches to synchronize before starting the worker
	if !cache.WaitForCacheSync(stopChan, c.controller.HasSynced) {
		runtime.HandleError(errors.New("Timed out waiting for cache to sync in scheduling manager"))
		return
	}

	c.worker.Run(stopChan)

	go func() {
		<-stopChan
		c.shutdown()
	}()
}

func (c *SchedulingManager) shutdown() {
	for _, scheduler := range c.schedulers.GetAll() {
		// Indicate scheduler and associated goroutines to be stopped in schedulingpreference controller.
		close(scheduler.(*SchedulerWrapper).stopChan)
	}
}

func (c *SchedulingManager) reconcile(qualifiedName util.QualifiedName) util.ReconciliationStatus {
	key := qualifiedName.String()

	klog.V(3).Infof("Running reconcile FederatedTypeConfig %q in scheduling manager", key)

	typeConfigName := qualifiedName.Name
	schedulingType := schedulingtypes.GetSchedulingType(typeConfigName)
	if schedulingType == nil {
		// No scheduler supported for this resource
		return util.StatusAllOK
	}
	schedulingKind := schedulingType.Kind

	cachedObj, exist, err := c.store.GetByKey(key)
	if err != nil {
		runtime.HandleError(errors.Wrapf(err, "Failed to query FederatedTypeConfig store for %q in scheduling manager", key))
		return util.StatusError
	}

	if !exist {
		c.stopScheduler(schedulingKind, typeConfigName)
		return util.StatusAllOK
	}

	typeConfig := cachedObj.(*corev1b1.FederatedTypeConfig)
	if !typeConfig.GetPropagationEnabled() || typeConfig.DeletionTimestamp != nil {
		c.stopScheduler(schedulingKind, typeConfigName)
		return util.StatusAllOK
	}

	// set name and group for the type config target
	corev1b1.SetFederatedTypeConfigDefaults(typeConfig)

	// Scheduling preference controller is started on demand
	abstractScheduler, ok := c.schedulers.Get(schedulingKind)
	if !ok {
		klog.Infof("Starting schedulingpreference controller for %s", schedulingKind)
		stopChan := make(chan struct{})
		schedulerInterface, err := schedulingpreference.StartSchedulingPreferenceController(c.config, *schedulingType, stopChan)
		if err != nil {
			runtime.HandleError(errors.Wrapf(err, "Error starting schedulingpreference controller for %s", schedulingKind))
			return util.StatusError
		}
		abstractScheduler = newSchedulerWrapper(schedulerInterface, stopChan)
		c.schedulers.Store(schedulingKind, abstractScheduler)
	}

	scheduler := abstractScheduler.(*SchedulerWrapper)
	if scheduler.HasPlugin(typeConfigName) {
		// Scheduler and plugin already running for this target typeConfig
		return util.StatusAllOK
	}

	federatedKind := typeConfig.GetFederatedType().Kind
	klog.Infof("Starting plugin %s for %s", federatedKind, schedulingKind)
	err = scheduler.StartPlugin(typeConfig)
	if err != nil {
		runtime.HandleError(errors.Wrapf(err, "Error starting plugin %s for %s", federatedKind, schedulingKind))
		return util.StatusError
	}
	scheduler.pluginMap.Store(typeConfigName, federatedKind)

	return util.StatusAllOK
}

func (c *SchedulingManager) stopScheduler(schedulingKind, typeConfigName string) {
	abstractScheduler, ok := c.schedulers.Get(schedulingKind)
	if !ok {
		return
	}

	scheduler := abstractScheduler.(*SchedulerWrapper)
	if scheduler.HasPlugin(typeConfigName) {
		kind, _ := scheduler.pluginMap.Get(typeConfigName)
		klog.Infof("Stopping plugin %s for %s", kind.(string), schedulingKind)
		scheduler.StopPlugin(kind.(string))
		scheduler.pluginMap.Delete(typeConfigName)
	}

	// If all plugins associated with this scheduler are gone, the scheduler should also be stopped.
	if scheduler.pluginMap.Size() == 0 {
		klog.Infof("Stopping schedulingpreference controller for %q", schedulingKind)
		// Indicate scheduler and associated goroutines to be stopped in schedulingpreference controller.
		close(scheduler.stopChan)
		c.schedulers.Delete(schedulingKind)
	}
}
