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

package sync

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/marun/federation-v2/pkg/apis/federation/common"
	fedv1a1 "github.com/marun/federation-v2/pkg/apis/federation/v1alpha1"
	fedclientset "github.com/marun/federation-v2/pkg/client/clientset_generated/clientset"
	"github.com/marun/federation-v2/pkg/controller/sync/placement"
	"github.com/marun/federation-v2/pkg/controller/util"
	"github.com/marun/federation-v2/pkg/controller/util/deletionhelper"
	"github.com/marun/federation-v2/pkg/federatedtypes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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

	// Store for the override directives of the federated type
	overrideStore cache.Store
	// Informer controller for override directives of the federated type
	overrideController cache.Controller

	placementPlugin placement.PlacementPlugin

	// Store for propagated versions
	propagatedVersionStore cache.Store
	// Informer for propagated versions
	propagatedVersionController cache.Controller
	// Map of keys to versions for pending updates
	pendingVersionUpdates sets.String
	// Helper for propagated version comparison for resource types.
	comparisonHelper util.ComparisonHelper

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

	adapter federatedtypes.FederatedTypeAdapter
}

// StartFederationSyncController starts a new sync controller for a type adapter
func StartFederationSyncController(kind string, adapterFactory federatedtypes.AdapterFactory, fedConfig, kubeConfig, crConfig *restclient.Config, stopChan <-chan struct{}, minimizeLatency bool) error {
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
	namespaceAdapter, ok := adapter.(*federatedtypes.FederatedNamespaceAdapter)
	if ok {
		namespaceAdapter.SetKubeClient(kubeClient)
	}
	controller, err := newFederationSyncController(adapter, fedConfig, fedClient, kubeClient, crClient)
	if err != nil {
		return err
	}
	if minimizeLatency {
		controller.minimizeLatency()
	}
	glog.Infof(fmt.Sprintf("Starting sync controller for %s resources", kind))
	controller.Run(stopChan)
	return nil
}

// newFederationSyncController returns a new sync controller for the given client and type adapter
func newFederationSyncController(adapter federatedtypes.FederatedTypeAdapter, fedConfig *restclient.Config, fedClient fedclientset.Interface, kubeClient kubeclientset.Interface, crClient crclientset.Interface) (*FederationSyncController, error) {
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
		pendingVersionUpdates:   sets.String{},
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

	override := adapter.Override()
	if override != nil {
		s.overrideStore, s.overrideController = newFedApiInformer(adapter.Override(), deliverObj)
	}

	deliverUnstructured := func(obj *unstructured.Unstructured) {
		qualifiedName := federatedtypes.QualifiedName{
			Namespace: obj.GetNamespace(),
			Name:      obj.GetName(),
		}
		s.deliver(qualifiedName, 0, false)
	}
	if adapter.Template().Kind() == federatedtypes.NamespaceKind {
		s.placementPlugin = placement.NewNamespacePlacementPlugin(adapter.Placement(), deliverObj)
	} else {
		var err error
		s.placementPlugin, err = placement.NewResourcePlacementPlugin(fedConfig, adapter.PlacementGroupVersionResource(), deliverUnstructured)
		if err != nil {
			return nil, err
		}
	}

	s.propagatedVersionStore, s.propagatedVersionController = cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (pkgruntime.Object, error) {
				return fedClient.FederationV1alpha1().PropagatedVersions(metav1.NamespaceAll).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return fedClient.FederationV1alpha1().PropagatedVersions(metav1.NamespaceAll).Watch(options)
			},
		},
		&fedv1a1.PropagatedVersion{},
		util.NoResyncPeriod,
		&cache.ResourceEventHandlerFuncs{
			DeleteFunc: func(old interface{}) {},
			AddFunc:    func(cur interface{}) {},
			UpdateFunc: func(old, cur interface{}) {
				// Clear the indication of a pending version update for the object's key
				version := cur.(*fedv1a1.PropagatedVersion)
				key := federatedtypes.NewQualifiedName(version).String()
				// TODO(marun) LOCK!
				if s.pendingVersionUpdates.Has(key) {
					s.pendingVersionUpdates.Delete(key)
				}
			},
		},
	)

	targetAdapter := adapter.Target()
	var err error
	s.comparisonHelper, err = util.NewComparisonHelper(targetAdapter.VersionCompareType())
	if err != nil {
		return nil, err
	}

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
		func(client kubeclientset.Interface, obj pkgruntime.Object) (string, error) {
			createdObj, err := targetAdapter.Create(client, obj)
			return s.comparisonHelper.GetVersion(targetAdapter.ObjectMeta(createdObj)), err
		},
		func(client kubeclientset.Interface, obj pkgruntime.Object) (string, error) {
			updatedObj, err := targetAdapter.Update(client, obj)
			return s.comparisonHelper.GetVersion(targetAdapter.ObjectMeta(updatedObj)), err
		},
		func(client kubeclientset.Interface, obj pkgruntime.Object) (string, error) {
			qualifiedName := federatedtypes.NewQualifiedName(obj)
			orphanDependents := false
			return "", targetAdapter.Delete(client, qualifiedName, &metav1.DeleteOptions{OrphanDependents: &orphanDependents})
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

	return s, nil
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
	if s.overrideController != nil {
		go s.overrideController.Run(stopChan)
	}
	go s.propagatedVersionController.Run(stopChan)
	go s.placementPlugin.Run(stopChan)
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
	if !s.placementPlugin.HasSynced() {
		glog.V(2).Infof("Placement not synced")
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
	namespace := qualifiedName.Namespace

	if federatedtypes.IsNamespaceKind(kind) {
		namespace = qualifiedName.Name
		// TODO(font): Need a configurable or discoverable list of namespaces
		// to not propagate beyond just the default system namespaces e.g.
		// clusterregistry.
		if isSystemNamespace(namespace) {
			return statusAllOK
		}
	}

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
		err := s.delete(template, meta, kind, qualifiedName)
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

	selectedClusters, unselectedClusters, err := s.placementPlugin.ComputePlacement(key, clusterNames)
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to compute placement for %s %q: %v", kind, key, err))
		return statusError
	}

	var override pkgruntime.Object
	if s.overrideStore != nil {
		override, err = s.objFromCache(s.overrideStore, s.adapter.Override().Kind(), key)
		if err != nil {
			return statusError
		}
	}

	propagatedVersionKey := federatedtypes.QualifiedName{
		Namespace: namespace,
		Name:      s.versionName(qualifiedName.Name),
	}.String()
	// TODO(marun) LOCK!
	if s.pendingVersionUpdates.Has(propagatedVersionKey) {
		// A status update is pending
		return statusNeedsRecheck
	}
	propagatedVersionFromCache, err := s.objFromCache(s.propagatedVersionStore,
		"PropagatedVersion", propagatedVersionKey)
	if err != nil {
		return statusError
	}
	var propagatedVersion *fedv1a1.PropagatedVersion
	if propagatedVersionFromCache != nil {
		propagatedVersion = propagatedVersionFromCache.(*fedv1a1.PropagatedVersion)
	}

	return s.syncToClusters(selectedClusters, unselectedClusters, template, override, propagatedVersion)
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
func (s *FederationSyncController) delete(template pkgruntime.Object, meta *metav1.ObjectMeta,
	kind string, qualifiedName federatedtypes.QualifiedName) error {
	glog.V(3).Infof("Handling deletion of %s %q", kind, qualifiedName)

	_, err := s.deletionHelper.HandleObjectInUnderlyingClusters(template, meta, kind)
	if err != nil {
		return err
	}

	if federatedtypes.IsNamespaceKind(kind) {
		// Return immediately if we are a namespace as it will be deleted
		// simply by removing its finalizers.
		return nil
	}

	versionName := s.versionName(qualifiedName.Name)
	err = s.adapter.FedClient().FederationV1alpha1().PropagatedVersions(qualifiedName.Namespace).Delete(versionName, nil)
	if err != nil && !errors.IsNotFound(err) {
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

func (s *FederationSyncController) versionName(resourceName string) string {
	targetKind := s.adapter.Target().Kind()
	return common.PropagatedVersionName(targetKind, resourceName)
}

// syncToClusters ensures that the state of the given object is synchronized to
// member clusters.
func (s *FederationSyncController) syncToClusters(selectedClusters, unselectedClusters []string,
	template, override pkgruntime.Object, propagatedVersion *fedv1a1.PropagatedVersion) reconciliationStatus {

	kind := s.adapter.Template().Kind()
	key := federatedtypes.NewQualifiedName(template).String()

	glog.V(3).Infof("Syncing %s %q in underlying clusters", kind, key)

	propagatedClusterVersions := getClusterVersions(s.adapter, template, override, propagatedVersion)

	operations, err := s.clusterOperations(selectedClusters, unselectedClusters, template,
		override, key, propagatedClusterVersions)
	if err != nil {
		s.eventRecorder.Eventf(template, corev1.EventTypeWarning, "FedClusterOperationsError",
			"Error obtaining sync operations for %s: %s error: %s", kind, key, err.Error())
		return statusError
	}

	if len(operations) == 0 {
		return statusAllOK
	}

	updatedClusterVersions, operationErrors := s.updater.Update(operations)

	err = updatePropagatedVersion(s.adapter, updatedClusterVersions, template, override,
		propagatedVersion, selectedClusters, s.pendingVersionUpdates)
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to record propagated version for %s %q: %v", kind,
			key, err))
		// Don't return an error - failure to record the propagated
		// version does not imply that propagation for the resource
		// needs to be attempted again.
	}

	if len(operationErrors) > 0 {
		runtime.HandleError(fmt.Errorf("Failed to execute updates for %s %q: %v", kind,
			key, operationErrors))
		return statusError
	}

	return statusAllOK
}

// getClusterVersions returns the cluster versions populated in the current
// propagated version object if it exists.
func getClusterVersions(adapter federatedtypes.FederatedTypeAdapter, template,
	override pkgruntime.Object, propagatedVersion *fedv1a1.PropagatedVersion) map[string]string {

	clusterVersions := make(map[string]string)

	if propagatedVersion == nil {
		return clusterVersions
	}

	templateVersion := adapter.Template().ObjectMeta(template).ResourceVersion
	overrideVersion := ""

	if override != nil {
		overrideVersion = adapter.Override().ObjectMeta(override).ResourceVersion
	}

	if templateVersion == propagatedVersion.Status.TemplateVersion &&
		overrideVersion == propagatedVersion.Status.OverrideVersion {
		for _, versions := range propagatedVersion.Status.ClusterVersions {
			clusterVersions[versions.ClusterName] = versions.Version
		}
	}

	return clusterVersions
}

// updatePropagatedVersion handles creating or updating the propagated version
// resource in the federation API.
func updatePropagatedVersion(adapter federatedtypes.FederatedTypeAdapter,
	updatedVersions map[string]string, template, override pkgruntime.Object,
	version *fedv1a1.PropagatedVersion, selectedClusters []string,
	pendingVersionUpdates sets.String) error {

	templateMeta := adapter.Template().ObjectMeta(template)
	overrideVersion := ""
	if override != nil {
		overrideVersion = adapter.Override().ObjectMeta(override).ResourceVersion
	}

	if version == nil {
		version := newVersion(updatedVersions, templateMeta, adapter.Target().Kind(), overrideVersion)
		_, err := adapter.FedClient().FederationV1alpha1().PropagatedVersions(version.Namespace).Create(version)
		if err != nil {
			return err
		}

		key := federatedtypes.NewQualifiedName(version).String()
		// TODO(marun) add timeout to ensure against lost updates blocking propagation of a given resource
		pendingVersionUpdates.Insert(key)

		_, err = adapter.FedClient().FederationV1alpha1().PropagatedVersions(version.Namespace).UpdateStatus(version)
		if err != nil {
			pendingVersionUpdates.Delete(key)
		}
		return err
	}

	oldVersionStatus := version.Status
	templateVersion := templateMeta.ResourceVersion
	var existingVersions []fedv1a1.ClusterObjectVersion
	if version.Status.TemplateVersion == templateVersion && version.Status.OverrideVersion == overrideVersion {
		existingVersions = version.Status.ClusterVersions
	} else {
		version.Status.TemplateVersion = templateVersion
		version.Status.OverrideVersion = overrideVersion
		existingVersions = []fedv1a1.ClusterObjectVersion{}
	}
	version.Status.ClusterVersions = clusterVersions(updatedVersions, existingVersions, selectedClusters)

	if util.PropagatedVersionStatusEquivalent(&oldVersionStatus, &version.Status) {
		glog.V(4).Infof("No PropagatedVersion update necessary for %s %q",
			adapter.Template().Kind(), federatedtypes.NewQualifiedName(template).String())
		return nil
	}

	key := federatedtypes.NewQualifiedName(version).String()
	pendingVersionUpdates.Insert(key)

	_, err := adapter.FedClient().FederationV1alpha1().PropagatedVersions(version.Namespace).UpdateStatus(version)
	if err != nil {
		pendingVersionUpdates.Delete(key)
	}
	return err
}

// newVersion initializes a new propagated version resource for the given
// cluster versions and template and override.
func newVersion(clusterVersions map[string]string, templateMeta *metav1.ObjectMeta, targetKind,
	overrideVersion string) *fedv1a1.PropagatedVersion {
	versions := []fedv1a1.ClusterObjectVersion{}
	for clusterName, version := range clusterVersions {
		versions = append(versions, fedv1a1.ClusterObjectVersion{
			ClusterName: clusterName,
			Version:     version,
		})
	}

	util.SortClusterVersions(versions)
	var namespace string
	if federatedtypes.IsNamespaceKind(targetKind) {
		namespace = templateMeta.Name
	} else {
		namespace = templateMeta.Namespace
	}

	return &fedv1a1.PropagatedVersion{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      common.PropagatedVersionName(targetKind, templateMeta.Name),
		},
		Status: fedv1a1.PropagatedVersionStatus{
			TemplateVersion: templateMeta.ResourceVersion,
			OverrideVersion: overrideVersion,
			ClusterVersions: versions,
		},
	}
}

// clusterVersions updates the versions for the given status.
func clusterVersions(newVersions map[string]string, oldVersions []fedv1a1.ClusterObjectVersion,
	selectedClusters []string) []fedv1a1.ClusterObjectVersion {
	// Retain versions for selected clusters that were not changed
	selectedClusterSet := sets.NewString(selectedClusters...)
	for _, oldVersion := range oldVersions {
		if !selectedClusterSet.Has(oldVersion.ClusterName) {
			continue
		}
		if _, ok := newVersions[oldVersion.ClusterName]; !ok {
			newVersions[oldVersion.ClusterName] = oldVersion.Version
		}
	}

	// Convert map to slice
	versions := []fedv1a1.ClusterObjectVersion{}
	for clusterName, version := range newVersions {
		// Lack of version indicates deletion
		if version == "" {
			continue
		}
		versions = append(versions, fedv1a1.ClusterObjectVersion{
			ClusterName: clusterName,
			Version:     version,
		})
	}

	util.SortClusterVersions(versions)
	return versions
}

// clusterOperations returns the list of operations needed to synchronize the
// state of the given object to the provided clusters.
func (s *FederationSyncController) clusterOperations(selectedClusters, unselectedClusters []string,
	template, override pkgruntime.Object, key string,
	clusterVersions map[string]string) ([]util.FederatedOperation, error) {

	operations := make([]util.FederatedOperation, 0)

	kind := s.adapter.Target().Kind()
	for _, clusterName := range selectedClusters {
		desiredObj := s.adapter.ObjectForCluster(template, override, clusterName)

		clusterObj, found, err := s.informer.GetTargetStore().GetByKey(clusterName, key)
		if err != nil {
			wrappedErr := fmt.Errorf("Failed to get %s %q from cluster %q: %v", kind, key, clusterName, err)
			runtime.HandleError(wrappedErr)
			return nil, wrappedErr
		}

		var operationType util.FederatedOperationType = ""

		if found {
			clusterObj := clusterObj.(pkgruntime.Object)

			// If we're a namespace kind and this is an object for the primary
			// cluster, then skip the version comparison check as we do not
			// track the cluster version for namespaces in the primary cluster.
			// This avoids unnecessary updates that triggers an infinite loop
			// of continually adding finalizers and then removing finalizers,
			// causing PropagatedVersion to not keep up with the
			// ResourceVersions being updated.
			if federatedtypes.IsNamespaceKind(kind) {
				templateMeta := s.adapter.Template().ObjectMeta(template)
				clusterMeta := s.adapter.Target().ObjectMeta(clusterObj)
				if util.IsPrimaryCluster(templateMeta, clusterMeta) {
					continue
				}
			}

			desiredObj = s.adapter.ObjectForUpdateOp(desiredObj, clusterObj)

			version, ok := clusterVersions[clusterName]
			if !ok {
				// No target version recorded for template+override version
				operationType = util.OperationTypeUpdate
			} else {
				clusterObjMeta := s.adapter.Target().ObjectMeta(clusterObj)
				desiredObjMeta := s.adapter.Target().ObjectMeta(desiredObj)
				targetVersion := s.comparisonHelper.GetVersion(clusterObjMeta)

				// Check if versions don't match. If they match then check its
				// ObjectMeta which only applies to resources where Generation
				// is used to track versions because Generation is only updated
				// when Spec changes.
				if version != targetVersion {
					operationType = util.OperationTypeUpdate
				} else if !s.comparisonHelper.Equivalent(desiredObjMeta, clusterObjMeta) {
					operationType = util.OperationTypeUpdate
				}
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
		clusterObj, found, err := s.informer.GetTargetStore().GetByKey(clusterName, key)
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

func newFedApiInformer(typeAdapter federatedtypes.FedApiAdapter, triggerFunc func(pkgruntime.Object)) (cache.Store, cache.Controller) {
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

// TODO (font): Externalize this list to a package var to allow it to be
// configurable.
func isSystemNamespace(namespace string) bool {
	switch namespace {
	case "kube-system", util.FederationSystemNamespace:
		return true
	default:
		return false
	}
}
