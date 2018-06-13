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

package manager

// TODO(marun) Rewrite with kubebuilder framework

// import (
// 	"fmt"
// 	"log"

// 	"github.com/golang/glog"

// 	"github.com/kubernetes-incubator/apiserver-builder/pkg/builders"
// 	"github.com/kubernetes-incubator/apiserver-builder/pkg/controller"

// 	"github.com/kubernetes-sigs/federation-v2/pkg/apis/federation/v1alpha1"
// 	"github.com/kubernetes-sigs/federation-v2/pkg/client/clientset_generated/clientset"
// 	listers "github.com/kubernetes-sigs/federation-v2/pkg/client/listers_generated/federation/v1alpha1"
// 	"github.com/kubernetes-sigs/federation-v2/pkg/controller/sharedinformers"
// 	"github.com/kubernetes-sigs/federation-v2/pkg/controller/sync"
// 	"k8s.io/apimachinery/pkg/api/errors"
// 	"k8s.io/apimachinery/pkg/api/meta"
// 	"k8s.io/apimachinery/pkg/util/sets"
// 	"k8s.io/client-go/rest"
// 	"k8s.io/client-go/tools/cache"
// 	"k8s.io/client-go/util/workqueue"
// )

// const (
// 	FinalizerControllerManager string = "federation.k8s.io/controller-manager"
// )

// // FederatedTypeConfigController implements the controller.FederatedTypeConfigController interface
// type FederatedTypeConfigController struct {
// 	queue *controller.QueueWorker

// 	// Handles messages
// 	controller *FederatedTypeConfigControllerImpl

// 	Name string

// 	BeforeReconcile func(key string)
// 	AfterReconcile  func(key string, err error)

// 	Informers *sharedinformers.SharedInformers
// }

// // NewController returns a new FederatedTypeConfigController for responding to FederatedTypeConfig events
// func NewFederatedTypeConfigController(config *rest.Config, si *sharedinformers.SharedInformers) *FederatedTypeConfigController {
// 	q := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "FederatedTypeConfig")

// 	queue := &controller.QueueWorker{q, 10, "FederatedTypeConfig", nil}
// 	c := &FederatedTypeConfigController{queue, nil, "FederatedTypeConfig", nil, nil, si}

// 	// For non-generated code to add events
// 	uc := &FederatedTypeConfigControllerImpl{
// 		config: config,
// 	}
// 	var ci sharedinformers.Controller = uc

// 	// Call the Init method that is implemented.
// 	// Support multiple Init methods for backwards compatibility
// 	if i, ok := ci.(sharedinformers.LegacyControllerInit); ok {
// 		i.Init(config, si, c.LookupAndReconcile)
// 	} else if i, ok := ci.(sharedinformers.ControllerInit); ok {
// 		i.Init(&sharedinformers.ControllerInitArgumentsImpl{si, config, c.LookupAndReconcile})
// 	}

// 	c.controller = uc

// 	queue.Reconcile = c.reconcile
// 	if c.Informers.WorkerQueues == nil {
// 		c.Informers.WorkerQueues = map[string]*controller.QueueWorker{}
// 	}
// 	c.Informers.WorkerQueues["FederatedTypeConfig"] = queue
// 	si.Factory.Federation().V1alpha1().FederatedTypeConfigs().Informer().
// 		AddEventHandler(&controller.QueueingEventHandler{q, nil, false})
// 	return c
// }

// func (c *FederatedTypeConfigController) GetName() string {
// 	return c.Name
// }

// func (c *FederatedTypeConfigController) LookupAndReconcile(key string) (err error) {
// 	return c.reconcile(key)
// }

// func (c *FederatedTypeConfigController) reconcile(key string) (err error) {
// 	var namespace, name string

// 	if c.BeforeReconcile != nil {
// 		c.BeforeReconcile(key)
// 	}
// 	if c.AfterReconcile != nil {
// 		// Wrap in a function so err is evaluated after it is set
// 		defer func() { c.AfterReconcile(key, err) }()
// 	}

// 	namespace, name, err = cache.SplitMetaNamespaceKey(key)
// 	if err != nil {
// 		return
// 	}

// 	u, err := c.controller.Get(namespace, name)
// 	if errors.IsNotFound(err) {
// 		glog.Infof("Not doing work for FederatedTypeConfig %v because it has been deleted", key)
// 		// Set error so it is picked up by AfterReconcile and the return function
// 		err = nil
// 		return
// 	}
// 	if err != nil {
// 		glog.Errorf("Unable to retrieve FederatedTypeConfig %v from store: %v", key, err)
// 		return
// 	}

// 	// Set error so it is picked up by AfterReconcile and the return function
// 	err = c.controller.Reconcile(u)

// 	return
// }

// func (c *FederatedTypeConfigController) Run(stopCh <-chan struct{}) {
// 	for _, q := range c.Informers.WorkerQueues {
// 		q.Run(stopCh)
// 	}
// 	controller.GetDefaults(c.controller).Run(stopCh)
// 	// Ensure that the internal controller gets shutdown
// 	go func() {
// 		<-stopCh
// 		c.controller.ShutDown()
// 	}()
// }

// type FederatedTypeConfigControllerImpl struct {
// 	builders.DefaultControllerFns

// 	// lister indexes properties about FederatedTypeConfig
// 	lister listers.FederatedTypeConfigLister

// 	// Need config reference to enable instantiation of new sync controllers
// 	config *rest.Config

// 	// Client for updates
// 	client clientset.Interface

// 	// Map of running sync controllers keyed by qualified target type
// 	//
// 	// TODO(marun) Does access to this map need to be synchronized?
// 	// It's not clear whether Reconcile can be potentially called from
// 	// more than one thread.
// 	stopChannels map[string]chan struct{}
// }

// // Init initializes the controller and is called by the generated code
// // Register watches for additional resource types here.
// func (c *FederatedTypeConfigControllerImpl) Init(arguments sharedinformers.ControllerInitArguments) {
// 	// Use the lister for indexing federatedtypeconfigs labels
// 	c.lister = arguments.GetSharedInformers().Factory.Federation().V1alpha1().FederatedTypeConfigs().Lister()
// 	c.stopChannels = make(map[string]chan struct{})
// }

// // Reconcile handles enqueued messages
// func (c *FederatedTypeConfigControllerImpl) Reconcile(u *v1alpha1.FederatedTypeConfig) error {
// 	log.Printf("Running reconcile FederatedTypeConfig for %s\n", u.Name)

// 	stopChan, running := c.stopChannels[u.Name]

// 	deleted := u.DeletionTimestamp != nil
// 	if deleted {
// 		if running {
// 			c.stopController(u.Name, stopChan)
// 		}
// 		return c.removeFinalizer(u)
// 	}

// 	err := c.ensureFinalizer(u)
// 	if err != nil {
// 		return err
// 	}

// 	enabled := u.Spec.PropagationEnabled
// 	startNewController := !running && enabled
// 	stopController := running && !enabled
// 	if startNewController {
// 		stopChan = make(chan struct{})
// 		err := sync.StartFederationSyncController(u, c.config, c.config, c.config, stopChan, false)
// 		if err != nil {
// 			close(stopChan)
// 			return fmt.Errorf("Error starting sync controller for %q: %v", u.Spec.Template.Kind, err)
// 		}
// 		c.stopChannels[u.Name] = stopChan
// 	} else if stopController {
// 		c.stopController(u.Name, stopChan)
// 	}

// 	return nil
// }

// func (c *FederatedTypeConfigControllerImpl) stopController(key string, stopChan chan struct{}) {
// 	log.Printf("Stopping sync controller for %s \n", key)
// 	close(stopChan)
// 	delete(c.stopChannels, key)
// }

// func (c *FederatedTypeConfigControllerImpl) ensureFinalizer(u *v1alpha1.FederatedTypeConfig) error {
// 	accessor, err := meta.Accessor(u)
// 	if err != nil {
// 		return err
// 	}
// 	finalizers := sets.NewString(accessor.GetFinalizers()...)
// 	if finalizers.Has(FinalizerControllerManager) {
// 		return nil
// 	}
// 	finalizers.Insert(FinalizerControllerManager)
// 	return c.update(u)
// }

// func (c *FederatedTypeConfigControllerImpl) removeFinalizer(u *v1alpha1.FederatedTypeConfig) error {
// 	accessor, err := meta.Accessor(u)
// 	if err != nil {
// 		return err
// 	}
// 	finalizers := sets.NewString(accessor.GetFinalizers()...)
// 	if !finalizers.Has(FinalizerControllerManager) {
// 		return nil
// 	}
// 	finalizers.Delete(FinalizerControllerManager)
// 	return c.update(u)
// }

// func (c *FederatedTypeConfigControllerImpl) update(u *v1alpha1.FederatedTypeConfig) error {
// 	if c.client == nil {
// 		c.client = clientset.NewForConfigOrDie(c.config)
// 	}
// 	_, err := c.client.CoreV1alpha1().FederatedTypeConfigs().Update(u)
// 	return err
// }

// func (c *FederatedTypeConfigControllerImpl) Get(namespace, name string) (*v1alpha1.FederatedTypeConfig, error) {
// 	return c.lister.Get(name)
// }

// func (c *FederatedTypeConfigControllerImpl) ShutDown() {
// 	// Stop all sync controllers
// 	for key, stopChannel := range c.stopChannels {
// 		close(stopChannel)
// 		delete(c.stopChannels, key)
// 	}
// }
