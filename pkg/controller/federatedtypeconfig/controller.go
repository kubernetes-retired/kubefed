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

package federatedtypeconfig

import (
	"context"
	"sync"

	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"

	corev1b1 "sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	genericclient "sigs.k8s.io/kubefed/pkg/client/generic"
	statuscontroller "sigs.k8s.io/kubefed/pkg/controller/status"
	synccontroller "sigs.k8s.io/kubefed/pkg/controller/sync"
	"sigs.k8s.io/kubefed/pkg/controller/util"
)

const finalizer string = "core.kubefed.io/federated-type-config"

// The FederatedTypeConfig controller configures sync and status
// controllers in response to FederatedTypeConfig resources in the
// KubeFed system namespace.
type Controller struct {
	// Arguments to use when starting new controllers
	controllerConfig *util.ControllerConfig

	client genericclient.Client

	// Map of running sync controllers keyed by qualified target type
	stopChannels map[string]chan struct{}
	lock         sync.RWMutex

	// Store for the FederatedTypeConfig objects
	store cache.Store
	// Informer for the FederatedTypeConfig objects
	controller cache.Controller

	worker util.ReconcileWorker
}

// StartController starts the Controller for managing FederatedTypeConfig objects.
func StartController(config *util.ControllerConfig, stopChan <-chan struct{}) error {
	controller, err := newController(config)
	if err != nil {
		return err
	}
	klog.Infof("Starting FederatedTypeConfig controller")
	controller.Run(stopChan)
	return nil
}

// newController returns a new controller to manage FederatedTypeConfig objects.
func newController(config *util.ControllerConfig) (*Controller, error) {
	userAgent := "FederatedTypeConfig"
	kubeConfig := restclient.CopyConfig(config.KubeConfig)
	restclient.AddUserAgent(kubeConfig, userAgent)
	genericclient, err := genericclient.New(kubeConfig)
	if err != nil {
		return nil, err
	}

	c := &Controller{
		controllerConfig: config,
		client:           genericclient,
		stopChannels:     make(map[string]chan struct{}),
	}

	c.worker = util.NewReconcileWorker(c.reconcile, util.WorkerTiming{})

	// Only watch the KubeFed namespace to ensure
	// restrictive authz can be applied to a namespaced
	// control plane.
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

// Run runs the Controller.
func (c *Controller) Run(stopChan <-chan struct{}) {
	go c.controller.Run(stopChan)

	// wait for the caches to synchronize before starting the worker
	if !cache.WaitForCacheSync(stopChan, c.controller.HasSynced) {
		runtime.HandleError(errors.New("Timed out waiting for cache to sync"))
		return
	}

	c.worker.Run(stopChan)

	// Ensure all goroutines are cleaned up when the stop channel closes
	go func() {
		<-stopChan
		c.shutDown()
	}()
}

func (c *Controller) reconcile(qualifiedName util.QualifiedName) util.ReconciliationStatus {
	key := qualifiedName.String()

	klog.V(3).Infof("Running reconcile FederatedTypeConfig for %q", key)

	cachedObj, err := c.objCopyFromCache(key)
	if err != nil {
		return util.StatusError
	}

	if cachedObj == nil {
		return util.StatusAllOK
	}
	typeConfig := cachedObj.(*corev1b1.FederatedTypeConfig)

	// TODO(marun) Perform this defaulting in a webhook
	corev1b1.SetFederatedTypeConfigDefaults(typeConfig)

	syncEnabled := typeConfig.GetPropagationEnabled()
	statusEnabled := typeConfig.GetStatusEnabled()

	limitedScope := c.controllerConfig.TargetNamespace != metav1.NamespaceAll
	if limitedScope && syncEnabled && !typeConfig.GetNamespaced() {
		_, ok := c.getStopChannel(typeConfig.Name)
		if !ok {
			holderChan := make(chan struct{})
			c.lock.Lock()
			c.stopChannels[typeConfig.Name] = holderChan
			c.lock.Unlock()
			klog.Infof("Skipping start of sync & status controller for cluster-scoped resource %q. It is not required for a namespaced KubeFed control plane.", typeConfig.GetFederatedType().Kind)
		}

		typeConfig.Status.ObservedGeneration = typeConfig.Generation
		typeConfig.Status.PropagationController = corev1b1.ControllerStatusNotRunning

		if typeConfig.Status.StatusController == nil {
			typeConfig.Status.StatusController = new(corev1b1.ControllerStatus)
		}
		*typeConfig.Status.StatusController = corev1b1.ControllerStatusNotRunning
		err = c.client.UpdateStatus(context.TODO(), typeConfig)
		if err != nil {
			runtime.HandleError(errors.Wrapf(err, "Could not update status fields of the CRD: %q", key))
			return util.StatusError
		}
		return util.StatusAllOK
	}

	statusKey := typeConfig.Name + "/status"
	syncStopChan, syncRunning := c.getStopChannel(typeConfig.Name)
	statusStopChan, statusRunning := c.getStopChannel(statusKey)

	deleted := typeConfig.DeletionTimestamp != nil
	if deleted {
		if syncRunning {
			c.stopController(typeConfig.Name, syncStopChan)
		}
		if statusRunning {
			c.stopController(statusKey, statusStopChan)
		}

		if typeConfig.IsNamespace() {
			klog.Infof("Reconciling all namespaced FederatedTypeConfig resources on deletion of %q", key)
			c.reconcileOnNamespaceFTCUpdate()
		}

		err := c.removeFinalizer(typeConfig)
		if err != nil {
			runtime.HandleError(errors.Wrapf(err, "Failed to remove finalizer from FederatedTypeConfig %q", key))
			return util.StatusError
		}
		return util.StatusAllOK
	}

	updated, err := c.ensureFinalizer(typeConfig)
	if err != nil {
		runtime.HandleError(errors.Wrapf(err, "Failed to ensure finalizer for FederatedTypeConfig %q", key))
		return util.StatusError
	} else if updated && typeConfig.IsNamespace() {
		// Detected creation of the namespace FTC. If there are existing FTCs
		// which did not start their sync controllers due to the lack of a
		// namespace FTC, then reconcile them now so they can start.
		klog.Infof("Reconciling all namespaced FederatedTypeConfig resources on finalizer update for %q", key)
		c.reconcileOnNamespaceFTCUpdate()
	}

	startNewSyncController := !syncRunning && syncEnabled
	stopSyncController := syncRunning && (!syncEnabled || (typeConfig.GetNamespaced() && !c.namespaceFTCExists()))
	if startNewSyncController {
		if err := c.startSyncController(typeConfig); err != nil {
			runtime.HandleError(err)
			return util.StatusError
		}
	} else if stopSyncController {
		c.stopController(typeConfig.Name, syncStopChan)
	}

	startNewStatusController := !statusRunning && statusEnabled
	stopStatusController := statusRunning && !statusEnabled
	if startNewStatusController {
		if err := c.startStatusController(statusKey, typeConfig); err != nil {
			runtime.HandleError(err)
			return util.StatusError
		}
	} else if stopStatusController {
		c.stopController(statusKey, statusStopChan)
	}

	if !startNewSyncController && !stopSyncController &&
		typeConfig.Status.ObservedGeneration != typeConfig.Generation {
		if err := c.refreshSyncController(typeConfig); err != nil {
			runtime.HandleError(err)
			return util.StatusError
		}
	}

	typeConfig.Status.ObservedGeneration = typeConfig.Generation
	syncControllerRunning := startNewSyncController || (syncRunning && !stopSyncController)
	if syncControllerRunning {
		typeConfig.Status.PropagationController = corev1b1.ControllerStatusRunning
	} else {
		typeConfig.Status.PropagationController = corev1b1.ControllerStatusNotRunning
	}

	if typeConfig.Status.StatusController == nil {
		typeConfig.Status.StatusController = new(corev1b1.ControllerStatus)
	}

	statusControllerRunning := startNewStatusController || (statusRunning && !stopStatusController)
	if statusControllerRunning {
		*typeConfig.Status.StatusController = corev1b1.ControllerStatusRunning
	} else {
		*typeConfig.Status.StatusController = corev1b1.ControllerStatusNotRunning
	}
	err = c.client.UpdateStatus(context.TODO(), typeConfig)
	if err != nil {
		runtime.HandleError(errors.Wrapf(err, "Could not update status fields of the CRD: %q", key))
		return util.StatusError
	}
	return util.StatusAllOK
}

func (c *Controller) objCopyFromCache(key string) (pkgruntime.Object, error) {
	cachedObj, exist, err := c.store.GetByKey(key)
	if err != nil {
		wrappedErr := errors.Wrapf(err, "Failed to query FederatedTypeConfig store for %q", key)
		runtime.HandleError(wrappedErr)
		return nil, err
	}
	if !exist {
		return nil, nil
	}
	return cachedObj.(pkgruntime.Object).DeepCopyObject(), nil
}

func (c *Controller) shutDown() {
	c.lock.Lock()
	defer c.lock.Unlock()

	// Stop all sync and status controllers
	for key, stopChannel := range c.stopChannels {
		close(stopChannel)
		delete(c.stopChannels, key)
	}
}

func (c *Controller) getStopChannel(name string) (chan struct{}, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	stopChan, ok := c.stopChannels[name]
	return stopChan, ok
}

func (c *Controller) startSyncController(tc *corev1b1.FederatedTypeConfig) error {
	// TODO(marun) Consider using a shared informer for federated
	// namespace that can be shared between all controllers of a
	// cluster-scoped KubeFed control plane.  A namespace-scoped
	// control plane would still have to use a non-shared informer due
	// to it not being possible to limit its scope.

	ftc := tc.DeepCopyObject().(*corev1b1.FederatedTypeConfig)
	kind := ftc.Spec.FederatedType.Kind

	// A sync controller for a namespaced resource must be supplied
	// with the ftc for namespaces so that it can consider federated
	// namespace placement when determining the placement for
	// contained resources.
	var fedNamespaceAPIResource *metav1.APIResource
	if ftc.GetNamespaced() {
		var err error
		fedNamespaceAPIResource, err = c.getFederatedNamespaceAPIResource()
		if err != nil {
			return errors.Wrapf(err, "Unable to start sync controller for %q due to missing FederatedTypeConfig for namespaces", kind)
		}
	}

	stopChan := make(chan struct{})
	err := synccontroller.StartKubeFedSyncController(c.controllerConfig, stopChan, ftc, fedNamespaceAPIResource)
	if err != nil {
		close(stopChan)
		return errors.Wrapf(err, "Error starting sync controller for %q", kind)
	}
	klog.Infof("Started sync controller for %q", kind)
	c.lock.Lock()
	defer c.lock.Unlock()
	c.stopChannels[ftc.Name] = stopChan
	return nil
}

func (c *Controller) startStatusController(statusKey string, tc *corev1b1.FederatedTypeConfig) error {
	kind := tc.Spec.FederatedType.Kind
	stopChan := make(chan struct{})
	ftc := tc.DeepCopyObject().(*corev1b1.FederatedTypeConfig)
	err := statuscontroller.StartKubeFedStatusController(c.controllerConfig, stopChan, ftc)
	if err != nil {
		close(stopChan)
		return errors.Wrapf(err, "Error starting status controller for %q", kind)
	}
	klog.Infof("Started status controller for %q", kind)
	c.lock.Lock()
	defer c.lock.Unlock()
	c.stopChannels[statusKey] = stopChan
	return nil
}

func (c *Controller) stopController(key string, stopChan chan struct{}) {
	klog.Infof("Stopping controller for %q", key)
	close(stopChan)
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.stopChannels, key)
}

func (c *Controller) refreshSyncController(tc *corev1b1.FederatedTypeConfig) error {
	klog.Infof("refreshing sync controller for %q", tc.Name)

	syncStopChan, ok := c.getStopChannel(tc.Name)
	if ok {
		c.stopController(tc.Name, syncStopChan)
	}

	return c.startSyncController(tc)
}

func (c *Controller) ensureFinalizer(tc *corev1b1.FederatedTypeConfig) (bool, error) {
	accessor, err := meta.Accessor(tc)
	if err != nil {
		return false, err
	}
	finalizers := sets.NewString(accessor.GetFinalizers()...)
	if finalizers.Has(finalizer) {
		return false, nil
	}
	finalizers.Insert(finalizer)
	accessor.SetFinalizers(finalizers.List())
	err = c.client.Update(context.TODO(), tc)
	return true, err
}

func (c *Controller) removeFinalizer(tc *corev1b1.FederatedTypeConfig) error {
	accessor, err := meta.Accessor(tc)
	if err != nil {
		return err
	}
	finalizers := sets.NewString(accessor.GetFinalizers()...)
	if !finalizers.Has(finalizer) {
		return nil
	}
	finalizers.Delete(finalizer)
	accessor.SetFinalizers(finalizers.List())
	err = c.client.Update(context.TODO(), tc)
	return err
}

func (c *Controller) namespaceFTCExists() bool {
	_, err := c.getFederatedNamespaceAPIResource()
	return err == nil
}

func (c *Controller) getFederatedNamespaceAPIResource() (*metav1.APIResource, error) {
	qualifiedName := util.QualifiedName{
		Namespace: c.controllerConfig.KubeFedNamespace,
		Name:      util.NamespaceName,
	}
	key := qualifiedName.String()
	cachedObj, exists, err := c.store.GetByKey(key)
	if err != nil {
		return nil, errors.Wrapf(err, "Error retrieving %q from the informer cache", key)
	}
	if !exists {
		return nil, errors.Errorf("Unable to find %q in the informer cache", key)
	}
	namespaceTypeConfig := cachedObj.(*corev1b1.FederatedTypeConfig)
	apiResource := namespaceTypeConfig.GetFederatedType()
	return &apiResource, nil
}

func (c *Controller) reconcileOnNamespaceFTCUpdate() {
	for _, cachedObj := range c.store.List() {
		typeConfig := cachedObj.(*corev1b1.FederatedTypeConfig)
		if typeConfig.GetNamespaced() && !typeConfig.IsNamespace() {
			c.worker.EnqueueObject(typeConfig)
		}
	}
}
