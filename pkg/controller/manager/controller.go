/*
Copyright 2018 The Federation v2 Authors.

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

// TODO(marun) It was necessary to copy the controller from
// pkg/controller/federatedtypeconfig to allow the rest config to be
// passed through to the impl type to enable sync controller
// instantiation.
// federatedtypeconfig.NewFederatedTypeConfigController is generated
// so modifying it directly was not an option.

// TODO(marun) make management of sync controllers thread safe

package manager

import (
	"fmt"
	"log"

	"github.com/golang/glog"

	"github.com/kubernetes-incubator/apiserver-builder/pkg/builders"
	"github.com/kubernetes-incubator/apiserver-builder/pkg/controller"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/federation/v1alpha1"
	listers "github.com/kubernetes-sigs/federation-v2/pkg/client/listers_generated/federation/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/sharedinformers"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/sync"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// FederatedTypeConfigController implements the controller.FederatedTypeConfigController interface
type FederatedTypeConfigController struct {
	queue *controller.QueueWorker

	// Handles messages
	controller *FederatedTypeConfigControllerImpl

	Name string

	BeforeReconcile func(key string)
	AfterReconcile  func(key string, err error)

	Informers *sharedinformers.SharedInformers
}

// NewController returns a new FederatedTypeConfigController for responding to FederatedTypeConfig events
func NewFederatedTypeConfigController(config *rest.Config, si *sharedinformers.SharedInformers) *FederatedTypeConfigController {
	q := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "FederatedTypeConfig")

	queue := &controller.QueueWorker{q, 10, "FederatedTypeConfig", nil}
	c := &FederatedTypeConfigController{queue, nil, "FederatedTypeConfig", nil, nil, si}

	// For non-generated code to add events
	uc := &FederatedTypeConfigControllerImpl{
		config: config,
	}
	var ci sharedinformers.Controller = uc

	// Call the Init method that is implemented.
	// Support multiple Init methods for backwards compatibility
	if i, ok := ci.(sharedinformers.LegacyControllerInit); ok {
		i.Init(config, si, c.LookupAndReconcile)
	} else if i, ok := ci.(sharedinformers.ControllerInit); ok {
		i.Init(&sharedinformers.ControllerInitArgumentsImpl{si, config, c.LookupAndReconcile})
	}

	c.controller = uc

	queue.Reconcile = c.reconcile
	if c.Informers.WorkerQueues == nil {
		c.Informers.WorkerQueues = map[string]*controller.QueueWorker{}
	}
	c.Informers.WorkerQueues["FederatedTypeConfig"] = queue
	si.Factory.Federation().V1alpha1().FederatedTypeConfigs().Informer().
		AddEventHandler(&controller.QueueingEventHandler{q, nil, false})
	return c
}

func (c *FederatedTypeConfigController) GetName() string {
	return c.Name
}

func (c *FederatedTypeConfigController) LookupAndReconcile(key string) (err error) {
	return c.reconcile(key)
}

func (c *FederatedTypeConfigController) reconcile(key string) (err error) {
	var namespace, name string

	if c.BeforeReconcile != nil {
		c.BeforeReconcile(key)
	}
	if c.AfterReconcile != nil {
		// Wrap in a function so err is evaluated after it is set
		defer func() { c.AfterReconcile(key, err) }()
	}

	namespace, name, err = cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return
	}

	u, err := c.controller.Get(namespace, name)
	if errors.IsNotFound(err) {
		glog.Infof("Not doing work for FederatedTypeConfig %v because it has been deleted", key)
		// Set error so it is picked up by AfterReconcile and the return function
		err = nil
		return
	}
	if err != nil {
		glog.Errorf("Unable to retrieve FederatedTypeConfig %v from store: %v", key, err)
		return
	}

	// Set error so it is picked up by AfterReconcile and the return function
	err = c.controller.Reconcile(u)

	return
}

func (c *FederatedTypeConfigController) Run(stopCh <-chan struct{}) {
	for _, q := range c.Informers.WorkerQueues {
		q.Run(stopCh)
	}
	controller.GetDefaults(c.controller).Run(stopCh)
	// Ensure that the internal controller gets shutdown
	go func() {
		<-stopCh
		c.controller.ShutDown()
	}()
}

type FederatedTypeConfigControllerImpl struct {
	builders.DefaultControllerFns

	// lister indexes properties about FederatedTypeConfig
	lister listers.FederatedTypeConfigLister

	// Need config reference to enable instantiation of new sync controllers
	config *rest.Config

	// Map of running sync controllers keyed by qualified target type
	stopChannels map[string]chan struct{}
}

// Init initializes the controller and is called by the generated code
// Register watches for additional resource types here.
func (c *FederatedTypeConfigControllerImpl) Init(arguments sharedinformers.ControllerInitArguments) {
	// Use the lister for indexing federatedtypeconfigs labels
	c.lister = arguments.GetSharedInformers().Factory.Federation().V1alpha1().FederatedTypeConfigs().Lister()
	c.stopChannels = make(map[string]chan struct{})
}

// Reconcile handles enqueued messages
func (c *FederatedTypeConfigControllerImpl) Reconcile(u *v1alpha1.FederatedTypeConfig) error {
	// Implement controller logic here
	log.Printf("Running reconcile FederatedTypeConfig for %s\n", u.Name)

	// TODO(marun) Add finalizer to ensure cleanup on deletion
	// TODO(marun) Indicate via a status change whether a controller is running

	// TODO(marun) Consider how to respond to changes other than enable/disable
	stopChan, running := c.stopChannels[u.Name]
	enabled := u.Spec.PropagationEnabled
	if enabled && !running {
		// Start new controller
		stopChan = make(chan struct{})
		err := sync.StartFederationSyncController(u, c.config, c.config, c.config, stopChan, false)
		if err != nil {
			close(stopChan)
			return fmt.Errorf("Error starting sync controller for %q: %v", u.Spec.Template.Kind, err)
		}
		c.stopChannels[u.Name] = stopChan
	} else if !enabled && running {
		// Stop controller
		close(stopChan)
		delete(c.stopChannels, u.Name)
	}

	return nil
}

func (c *FederatedTypeConfigControllerImpl) Get(namespace, name string) (*v1alpha1.FederatedTypeConfig, error) {
	return c.lister.Get(name)
}

func (c *FederatedTypeConfigControllerImpl) ShutDown() {
	// Stop all sync controllers
	for key, stopChannel := range c.stopChannels {
		close(stopChannel)
		delete(c.stopChannels, key)
	}
}
