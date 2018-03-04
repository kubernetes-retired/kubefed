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

package sync

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	fedv1a1 "github.com/marun/fnord/pkg/apis/federation/v1alpha1"
	fedclientset "github.com/marun/fnord/pkg/client/clientset_generated/clientset"
	"github.com/marun/fnord/pkg/controller/util"
	"github.com/marun/fnord/pkg/controller/util/deletionhelper"
	"github.com/marun/fnord/pkg/federatedtypes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/client-go/util/workqueue"
	crclientset "k8s.io/cluster-registry/pkg/client/clientset_generated/clientset"
)

const (
	allClustersKey = "ALL_CLUSTERS"
)

// FederationSyncController synchronizes the state of a federated type
// to clusters that are members of the federation.
type FederationSyncController struct {
	// For triggering reconciliation of a single resource. This is
	// used when there is an add/update/delete operation on a resource
	// in either federated API server or in some member of the
	// federation.
	deliverer *util.DelayingDeliverer

	// For triggering reconciliation of all target resources. This is
	// used when a new cluster becomes available.
	clusterDeliverer *util.DelayingDeliverer

	// Contains resources present in members of federation.
	informer util.FederatedInformer
	// For updating members of federation.
	updater util.FederatedUpdater

	// Store for the templates of the federated type
	templateStore cache.Store
	// Informer for the templates of the federated type
	templateController cache.Controller

	// Store for the placement directives of the federated type
	placementStore cache.Store
	// Informer controller for placement directives of the federated type
	placementController cache.Controller

	// Work queue allowing parallel processing of resources
	workQueue workqueue.Interface

	// Backoff manager
	backoff *flowcontrol.Backoff

	// For events
	eventRecorder record.EventRecorder

	deletionHelper *deletionhelper.DeletionHelper

	reviewDelay             time.Duration
	clusterAvailableDelay   time.Duration
	clusterUnavailableDelay time.Duration
	smallDelay              time.Duration
	updateTimeout           time.Duration

	adapter federatedtypes.PropagationAdapter
}

// StartFederationSyncController starts a new sync controller for a type adapter
func StartFederationSyncController(kind string, adapterFactory federatedtypes.AdapterFactory, fedConfig, kubeConfig, crConfig *restclient.Config, stopChan <-chan struct{}, minimizeLatency bool) {
	// TODO(marun) should there be a unified client rather than
	// requiring separate clients for fed and kube apis?  In an
	// aggregated scenario one config can drive both of them, but it
	// may be difficult to replicate that in integration testing.
	userAgent := fmt.Sprintf("%s-controller", kind)
	restclient.AddUserAgent(fedConfig, userAgent)
	fedClient := fedclientset.NewForConfigOrDie(fedConfig)
	restclient.AddUserAgent(kubeConfig, userAgent)
	kubeClient := kubeclientset.NewForConfigOrDie(kubeConfig)
	restclient.AddUserAgent(crConfig, userAgent)
	crClient := crclientset.NewForConfigOrDie(crConfig)
	adapter := adapterFactory(fedClient)
	controller := newFederationSyncController(adapter, fedClient, kubeClient, crClient)
	if minimizeLatency {
		controller.minimizeLatency()
	}
	glog.Infof(fmt.Sprintf("Starting sync controller for %s resources", kind))
	controller.Run(stopChan)
}

// newFederationSyncController returns a new sync controller for the given client and type adapter
func newFederationSyncController(adapter federatedtypes.PropagationAdapter, fedClient fedclientset.Interface, kubeClient kubeclientset.Interface, crClient crclientset.Interface) *FederationSyncController {
	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	recorder := broadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: fmt.Sprintf("%v-controller", adapter.Template().Kind())})

	s := &FederationSyncController{
		reviewDelay:             time.Second * 10,
		clusterAvailableDelay:   time.Second * 20,
		clusterUnavailableDelay: time.Second * 60,
		smallDelay:              time.Second * 3,
		updateTimeout:           time.Second * 30,
		workQueue:               workqueue.New(),
		backoff:                 flowcontrol.NewBackOff(5*time.Second, time.Minute),
		eventRecorder:           recorder,
		adapter:                 adapter,
	}

	// Build delivereres for triggering reconciliations.
	s.deliverer = util.NewDelayingDeliverer()
	s.clusterDeliverer = util.NewDelayingDeliverer()

	templateAdapter := adapter.Template()
	// Start informers on the resources for the federated type
	deliverObj := func(obj pkgruntime.Object) {
		s.deliverObj(obj, 0, false)
	}
	s.templateStore, s.templateController = newFedApiInformer(templateAdapter, deliverObj)
	s.placementStore, s.placementController = newFedApiInformer(adapter.Placement(), deliverObj)

	targetAdapter := adapter.Target()
	// Federated informer on the resource type in members of federation.
	s.informer = util.NewFederatedInformer(
		fedClient,
		kubeClient,
		crClient,
		func(cluster *fedv1a1.FederatedCluster, targetClient kubeclientset.Interface) (cache.Store, cache.Controller) {
			return cache.NewInformer(
				&cache.ListWatch{
					ListFunc: func(options metav1.ListOptions) (pkgruntime.Object, error) {
						return targetAdapter.List(targetClient, metav1.NamespaceAll, options)
					},
					WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
						return targetAdapter.Watch(targetClient, metav1.NamespaceAll, options)
					},
				},
				targetAdapter.ObjectType(),
				util.NoResyncPeriod,
				// Trigger reconciliation whenever something in federated cluster is changed. In most cases it
				// would be just confirmation that some operation on the target resource type had succeeded.
				util.NewTriggerOnAllChanges(
					func(obj pkgruntime.Object) {
						s.deliverObj(obj, s.reviewDelay, false)
					},
				))
		},

		&util.ClusterLifecycleHandlerFuncs{
			ClusterAvailable: func(cluster *fedv1a1.FederatedCluster) {
				// When new cluster becomes available process all the target resources again.
				s.clusterDeliverer.DeliverAt(allClustersKey, nil, time.Now().Add(s.clusterAvailableDelay))
			},
			// When a cluster becomes unavailable process all the target resources again.
			ClusterUnavailable: func(cluster *fedv1a1.FederatedCluster, _ []interface{}) {
				s.clusterDeliverer.DeliverAt(allClustersKey, nil, time.Now().Add(s.clusterUnavailableDelay))
			},
		},
	)

	// Federated updater along with Create/Update/Delete operations.
	s.updater = util.NewFederatedUpdater(s.informer, templateAdapter.Kind(), s.updateTimeout, s.eventRecorder, // non-federated
		func(client kubeclientset.Interface, obj pkgruntime.Object) error {
			_, err := targetAdapter.Create(client, obj)
			return err
		},
		func(client kubeclientset.Interface, obj pkgruntime.Object) error {
			_, err := targetAdapter.Update(client, obj)
			return err
		},
		func(client kubeclientset.Interface, obj pkgruntime.Object) error {
			qualifiedName := federatedtypes.NewQualifiedName(obj)
			orphanDependents := false
			err := targetAdapter.Delete(client, qualifiedName, &metav1.DeleteOptions{OrphanDependents: &orphanDependents})
			return err
		})

	s.deletionHelper = deletionhelper.NewDeletionHelper(
		templateAdapter.Update,
		// objNameFunc
		func(obj pkgruntime.Object) string {
			return federatedtypes.NewQualifiedName(obj).String()
		},
		s.informer,
		s.updater,
	)

	return s
}

// minimizeLatency reduces delays and timeouts to make the controller more responsive (useful for testing).
func (s *FederationSyncController) minimizeLatency() {
	s.clusterAvailableDelay = time.Second
	s.clusterUnavailableDelay = time.Second
	s.reviewDelay = 50 * time.Millisecond
	s.smallDelay = 20 * time.Millisecond
	s.updateTimeout = 5 * time.Second
}

func (s *FederationSyncController) Run(stopChan <-chan struct{}) {
	go s.templateController.Run(stopChan)
	go s.placementController.Run(stopChan)
	s.informer.Start()
	s.deliverer.StartWithHandler(func(item *util.DelayingDelivererItem) {
		s.workQueue.Add(item)
	})
	s.clusterDeliverer.StartWithHandler(func(_ *util.DelayingDelivererItem) {
		s.reconcileOnClusterChange()
	})

	// TODO: Allow multiple workers.
	go wait.Until(s.worker, time.Second, stopChan)

	util.StartBackoffGC(s.backoff, stopChan)

	// Ensure all goroutines are cleaned up when the stop channel closes
	go func() {
		<-stopChan
		s.informer.Stop()
		s.workQueue.ShutDown()
		s.deliverer.Stop()
		s.clusterDeliverer.Stop()
	}()
}

type reconciliationStatus int

const (
	statusAllOK reconciliationStatus = iota
	statusNeedsRecheck
	statusError
	statusNotSynced
)

func (s *FederationSyncController) worker() {
	for {
		obj, quit := s.workQueue.Get()
		if quit {
			return
		}

		item := obj.(*util.DelayingDelivererItem)
		qualifiedName := item.Value.(*federatedtypes.QualifiedName)
		status := s.reconcile(*qualifiedName)
		s.workQueue.Done(item)

		switch status {
		case statusAllOK:
			break
		case statusError:
			s.deliver(*qualifiedName, 0, true)
		case statusNeedsRecheck:
			s.deliver(*qualifiedName, s.reviewDelay, false)
		case statusNotSynced:
			s.deliver(*qualifiedName, s.clusterAvailableDelay, false)
		}
	}
}

func (s *FederationSyncController) deliverObj(obj pkgruntime.Object, delay time.Duration, failed bool) {
	qualifiedName := federatedtypes.NewQualifiedName(obj)
	s.deliver(qualifiedName, delay, failed)
}

// Adds backoff to delay if this delivery is related to some failure. Resets backoff if there was no failure.
func (s *FederationSyncController) deliver(qualifiedName federatedtypes.QualifiedName, delay time.Duration, failed bool) {
	key := qualifiedName.String()
	if failed {
		s.backoff.Next(key, time.Now())
		delay = delay + s.backoff.Get(key)
	} else {
		s.backoff.Reset(key)
	}
	s.deliverer.DeliverAfter(key, &qualifiedName, delay)
}

// Check whether all data stores are in sync. False is returned if any of the informer/stores is not yet
// synced with the corresponding api server.
func (s *FederationSyncController) isSynced() bool {
	if !s.informer.ClustersSynced() {
		glog.V(2).Infof("Cluster list not synced")
		return false
	}
	// TODO(marun) set clusters as ready in the test fixture?
	clusters, err := s.informer.GetReadyClusters()
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to get ready clusters: %v", err))
		return false
	}
	if !s.informer.GetTargetStore().ClustersSynced(clusters) {
		return false
	}
	return true
}

// The function triggers reconciliation of all target federated resources.
func (s *FederationSyncController) reconcileOnClusterChange() {
	if !s.isSynced() {
		s.clusterDeliverer.DeliverAt(allClustersKey, nil, time.Now().Add(s.clusterAvailableDelay))
	}
	for _, obj := range s.templateStore.List() {
		qualifiedName := federatedtypes.NewQualifiedName(obj.(pkgruntime.Object))
		s.deliver(qualifiedName, s.smallDelay, false)
	}
}

func (s *FederationSyncController) reconcile(qualifiedName federatedtypes.QualifiedName) reconciliationStatus {
	if !s.isSynced() {
		return statusNotSynced
	}

	templateAdapter := s.adapter.Template()
	kind := templateAdapter.Kind()
	key := qualifiedName.String()

	glog.V(4).Infof("Starting to reconcile %v %v", kind, key)
	startTime := time.Now()
	defer glog.V(4).Infof("Finished reconciling %v %v (duration: %v)", kind, key, time.Now().Sub(startTime))

	template, err := s.objFromCache(s.templateStore, kind, key)
	if err != nil {
		return statusError
	}
	if template == nil {
		return statusAllOK
	}

	meta := templateAdapter.ObjectMeta(template)
	if meta.DeletionTimestamp != nil {
		err := s.delete(template, kind, qualifiedName)
		if err != nil {
			msg := "Failed to delete %s %q: %v"
			args := []interface{}{kind, qualifiedName, err}
			runtime.HandleError(fmt.Errorf(msg, args...))
			s.eventRecorder.Eventf(template, corev1.EventTypeWarning, "DeleteFailed", msg, args...)
			return statusError
		}
		return statusAllOK
	}

	glog.V(3).Infof("Ensuring finalizers exist on %s %q", kind, key)
	template, err = s.deletionHelper.EnsureFinalizers(template)
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to ensure finalizers for %s %q: %v", kind, key, err))
		return statusError
	}

	clusterNames, err := s.clusterNames()
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to get cluster list: %v", err))
		return statusNotSynced
	}

	placement, err := s.objFromCache(s.placementStore, kind, key)
	if err != nil {
		return statusError
	}

	operationsAccessor := func(adapter federatedtypes.PropagationAdapter, selectedClusters, unselectedClusters []string, template, placement pkgruntime.Object) ([]util.FederatedOperation, error) {
		operations, err := clusterOperations(adapter, selectedClusters, unselectedClusters, template, key, func(clusterName string) (interface{}, bool, error) {
			return s.informer.GetTargetStore().GetByKey(clusterName, key)
		})
		if err != nil {
			s.eventRecorder.Eventf(template, corev1.EventTypeWarning, "FedClusterOperationsError", "Error obtaining sync operations for %s: %s error: %s", kind, key, err.Error())
		}
		return operations, err
	}

	return syncToClusters(
		operationsAccessor,
		clusterNames,
		s.updater.Update,
		s.adapter,
		s.informer,
		template,
		placement,
	)
}

func (s *FederationSyncController) objFromCache(store cache.Store, kind, key string) (pkgruntime.Object, error) {
	cachedObj, exist, err := store.GetByKey(key)
	if err != nil {
		wrappedErr := fmt.Errorf("Failed to query %s store for %q: %v", kind, key, err)
		runtime.HandleError(wrappedErr)
		return nil, err
	}
	if !exist {
		return nil, nil
	}
	return cachedObj.(pkgruntime.Object).DeepCopyObject(), nil
}

func (s *FederationSyncController) clusterNames() ([]string, error) {
	clusters, err := s.informer.GetReadyClusters()
	if err != nil {
		return nil, err
	}
	clusterNames := []string{}
	for _, cluster := range clusters {
		clusterNames = append(clusterNames, cluster.Name)
	}

	return clusterNames, nil
}

// delete deletes the given resource or returns error if the deletion was not complete.
func (s *FederationSyncController) delete(template pkgruntime.Object, kind string, qualifiedName federatedtypes.QualifiedName) error {
	glog.V(3).Infof("Handling deletion of %s %q", kind, qualifiedName)

	// TOD(marun) Perform pre-deletion cleanup for the namespace adapter

	_, err := s.deletionHelper.HandleObjectInUnderlyingClusters(template)
	if err != nil {
		return err
	}

	err = s.adapter.Template().Delete(qualifiedName, nil)
	if err != nil {
		// Its all good if the error is not found error. That means it is deleted already and we do not have to do anything.
		// This is expected when we are processing an update as a result of finalizer deletion.
		// The process that deleted the last finalizer is also going to delete the resource and we do not have to do anything.
		if !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

type operationsFunc func(federatedtypes.PropagationAdapter, []string, []string, pkgruntime.Object, pkgruntime.Object) ([]util.FederatedOperation, error)
type executionFunc func([]util.FederatedOperation) error

// syncToClusters ensures that the state of the given object is synchronized to member clusters.
func syncToClusters(operationsAccessor operationsFunc, clusterNames []string, execute executionFunc, adapter federatedtypes.PropagationAdapter, informer util.FederatedInformer, template, placement pkgruntime.Object) reconciliationStatus {
	kind := adapter.Template().Kind()
	key := federatedtypes.NewQualifiedName(template).String()

	glog.V(3).Infof("Syncing %s %q in underlying clusters", kind, key)

	selectedClusters, unselectedClusters := computePlacement(adapter.Placement(), placement, clusterNames)

	operations, err := operationsAccessor(adapter, selectedClusters, unselectedClusters, template, placement)
	if err != nil {
		return statusError
	}

	if len(operations) == 0 {
		return statusAllOK
	}

	err = execute(operations)
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to execute updates for %s %q: %v", kind, key, err))
		return statusError
	}

	// Everything is in order but let's be double sure
	return statusNeedsRecheck
}

type clusterObjectAccessorFunc func(clusterName string) (interface{}, bool, error)

// clusterOperations returns the list of operations needed to synchronize the state of the given object to the provided clusters
func clusterOperations(adapter federatedtypes.PropagationAdapter, selectedClusters, unselectedClusters []string, template pkgruntime.Object, key string, accessor clusterObjectAccessorFunc) ([]util.FederatedOperation, error) {
	operations := make([]util.FederatedOperation, 0)

	kind := adapter.Target().Kind()
	for _, clusterName := range selectedClusters {
		desiredObj := adapter.ObjectForCluster(template, clusterName)

		clusterObj, found, err := accessor(clusterName)
		if err != nil {
			wrappedErr := fmt.Errorf("Failed to get %s %q from cluster %q: %v", kind, key, clusterName, err)
			runtime.HandleError(wrappedErr)
			return nil, wrappedErr
		}

		var operationType util.FederatedOperationType = ""
		if found {
			clusterObj := clusterObj.(pkgruntime.Object)
			if !adapter.Target().Equivalent(desiredObj, clusterObj) { // non-federated
				operationType = util.OperationTypeUpdate
			}
		} else {
			operationType = util.OperationTypeAdd
		}

		if len(operationType) > 0 {
			operations = append(operations, util.FederatedOperation{
				Type:        operationType,
				Obj:         desiredObj,
				ClusterName: clusterName,
				Key:         key,
			})
		}
	}

	for _, clusterName := range unselectedClusters {
		clusterObj, found, err := accessor(clusterName)
		if err != nil {
			wrappedErr := fmt.Errorf("Failed to get %s %q from cluster %q: %v", kind, key, clusterName, err)
			runtime.HandleError(wrappedErr)
			return nil, wrappedErr
		}
		if found {
			operations = append(operations, util.FederatedOperation{
				Type:        util.OperationTypeDelete,
				Obj:         clusterObj.(pkgruntime.Object),
				ClusterName: clusterName,
				Key:         key,
			})
		}
	}

	return operations, nil
}

func newFedApiInformer(typeAdapter federatedtypes.FederationTypeAdapter, triggerFunc func(pkgruntime.Object)) (cache.Store, cache.Controller) {
	return cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (pkgruntime.Object, error) {
				return typeAdapter.List(metav1.NamespaceAll, options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return typeAdapter.Watch(metav1.NamespaceAll, options)
			},
		},
		typeAdapter.ObjectType(),
		util.NoResyncPeriod,
		util.NewTriggerOnAllChanges(triggerFunc),
	)
}

func computePlacement(adapter federatedtypes.PlacementAdapter, placement pkgruntime.Object, clusterNames []string) ([]string, []string) {
	clusterSet := sets.NewString(clusterNames...)
	selectedClusters := adapter.ClusterNames(placement)
	selectedClusterSet := sets.NewString(selectedClusters...)
	return clusterSet.Intersection(selectedClusterSet).List(), clusterSet.Difference(selectedClusterSet).List()
}
