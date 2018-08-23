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
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/common"
	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/sync/placement"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util/deletionhelper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/client-go/util/workqueue"
	crclientset "k8s.io/cluster-registry/pkg/client/clientset/versioned"
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

	typeConfig typeconfig.Interface

	fedClient      fedclientset.Interface
	templateClient util.ResourceClient

	fedNamespace string
}

// StartFederationSyncController starts a new sync controller for a type config
func StartFederationSyncController(typeConfig typeconfig.Interface, kubeConfig *restclient.Config, fedNamespace, clusterNamespace, targetNamespace string, stopChan <-chan struct{}, minimizeLatency bool) error {
	controller, err := newFederationSyncController(typeConfig, kubeConfig, fedNamespace, clusterNamespace, targetNamespace)
	if err != nil {
		return err
	}
	if minimizeLatency {
		controller.minimizeLatency()
	}
	glog.Infof(fmt.Sprintf("Starting sync controller for %s resources", typeConfig.GetTemplate().Kind))
	controller.Run(stopChan)
	return nil
}

// newFederationSyncController returns a new sync controller for the configuration
func newFederationSyncController(typeConfig typeconfig.Interface, kubeConfig *restclient.Config, fedNamespace, clusterNamespace, targetNamespace string) (*FederationSyncController, error) {
	templateAPIResource := typeConfig.GetTemplate()
	userAgent := fmt.Sprintf("%s-controller", strings.ToLower(templateAPIResource.Kind))
	// Initialize non-dynamic clients first to avoid polluting config
	restclient.AddUserAgent(kubeConfig, userAgent)
	fedClient := fedclientset.NewForConfigOrDie(kubeConfig)
	kubeClient := kubeclientset.NewForConfigOrDie(kubeConfig)
	crClient := crclientset.NewForConfigOrDie(kubeConfig)

	pool := dynamic.NewDynamicClientPool(kubeConfig)

	templateClient, err := util.NewResourceClient(pool, &templateAPIResource)
	if err != nil {
		return nil, err
	}

	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	recorder := broadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: userAgent})

	s := &FederationSyncController{
		reviewDelay:             time.Second * 10,
		clusterAvailableDelay:   time.Second * 20,
		clusterUnavailableDelay: time.Second * 60,
		smallDelay:              time.Second * 3,
		updateTimeout:           time.Second * 30,
		workQueue:               workqueue.New(),
		backoff:                 flowcontrol.NewBackOff(5*time.Second, time.Minute),
		eventRecorder:           recorder,
		typeConfig:              typeConfig,
		fedClient:               fedClient,
		templateClient:          templateClient,
		pendingVersionUpdates:   sets.String{},
		fedNamespace:            fedNamespace,
	}

	// Build delivereres for triggering reconciliations.
	s.deliverer = util.NewDelayingDeliverer()
	s.clusterDeliverer = util.NewDelayingDeliverer()

	// Start informers on the resources for the federated type
	deliverObj := func(obj pkgruntime.Object) {
		s.deliverObj(obj, 0, false)
	}
	s.templateStore, s.templateController = util.NewResourceInformer(templateClient, targetNamespace, deliverObj)

	if overrideAPIResource := typeConfig.GetOverride(); overrideAPIResource != nil {
		client, err := util.NewResourceClient(pool, overrideAPIResource)
		if err != nil {
			return nil, err
		}
		s.overrideStore, s.overrideController = util.NewResourceInformer(client, targetNamespace, deliverObj)
	}

	placementAPIResource := typeConfig.GetPlacement()
	placementClient, err := util.NewResourceClient(pool, &placementAPIResource)
	if err != nil {
		return nil, err
	}
	targetAPIResource := typeConfig.GetTarget()
	if targetAPIResource.Kind == util.NamespaceKind {
		s.placementPlugin = placement.NewNamespacePlacementPlugin(placementClient, deliverObj)
	} else {
		s.placementPlugin = placement.NewResourcePlacementPlugin(placementClient, deliverObj)
	}

	s.propagatedVersionStore, s.propagatedVersionController = cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (pkgruntime.Object, error) {
				return fedClient.CoreV1alpha1().PropagatedVersions(targetNamespace).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return fedClient.CoreV1alpha1().PropagatedVersions(targetNamespace).Watch(options)
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
				key := util.NewQualifiedName(version).String()
				// TODO(marun) LOCK!
				if s.pendingVersionUpdates.Has(key) {
					s.pendingVersionUpdates.Delete(key)
				}
			},
		},
	)

	s.comparisonHelper, err = util.NewComparisonHelper(typeConfig.GetComparisonField())
	if err != nil {
		return nil, err
	}

	// Federated informer on the resource type in members of federation.
	s.informer = util.NewFederatedInformer(
		fedClient,
		kubeClient,
		crClient,
		fedNamespace,
		clusterNamespace,
		targetNamespace,
		&targetAPIResource,
		func(obj pkgruntime.Object) {
			s.deliverObj(obj, s.reviewDelay, false)
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
	s.updater = util.NewFederatedUpdater(s.informer, targetAPIResource.Kind, s.updateTimeout, s.eventRecorder,
		func(client util.ResourceClient, rawObj pkgruntime.Object) (string, error) {
			obj := rawObj.(*unstructured.Unstructured)
			createdObj, err := client.Resources(obj.GetNamespace()).Create(obj)
			if err != nil {
				return "", err
			}
			return s.comparisonHelper.GetVersion(createdObj), err
		},
		func(client util.ResourceClient, rawObj pkgruntime.Object) (string, error) {
			obj := rawObj.(*unstructured.Unstructured)
			updatedObj, err := client.Resources(obj.GetNamespace()).Update(obj)
			if err != nil {
				return "", err
			}
			return s.comparisonHelper.GetVersion(updatedObj), err
		},
		func(client util.ResourceClient, obj pkgruntime.Object) (string, error) {
			qualifiedName := util.NewQualifiedName(obj)
			orphanDependents := false
			return "", client.Resources(qualifiedName.Namespace).Delete(qualifiedName.Name, &metav1.DeleteOptions{OrphanDependents: &orphanDependents})
		})

	// TODO(marun) - need to add finalizers to placement and overrides, too

	s.deletionHelper = deletionhelper.NewDeletionHelper(
		// updateObjFunc
		func(rawObj pkgruntime.Object) (pkgruntime.Object, error) {
			obj := rawObj.(*unstructured.Unstructured)
			return templateClient.Resources(obj.GetNamespace()).Update(obj)
		},
		// objNameFunc
		func(obj pkgruntime.Object) string {
			return util.NewQualifiedName(obj).String()
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

func (s *FederationSyncController) worker() {
	for {
		obj, quit := s.workQueue.Get()
		if quit {
			return
		}

		item := obj.(*util.DelayingDelivererItem)
		qualifiedName := item.Value.(*util.QualifiedName)
		status := s.reconcile(*qualifiedName)
		s.workQueue.Done(item)

		switch status {
		case util.StatusAllOK:
			break
		case util.StatusError:
			s.deliver(*qualifiedName, 0, true)
		case util.StatusNeedsRecheck:
			s.deliver(*qualifiedName, s.reviewDelay, false)
		case util.StatusNotSynced:
			s.deliver(*qualifiedName, s.clusterAvailableDelay, false)
		}
	}
}

func (s *FederationSyncController) deliverObj(obj pkgruntime.Object, delay time.Duration, failed bool) {
	qualifiedName := util.NewQualifiedName(obj)
	s.deliver(qualifiedName, delay, failed)
}

// Adds backoff to delay if this delivery is related to some failure. Resets backoff if there was no failure.
func (s *FederationSyncController) deliver(qualifiedName util.QualifiedName, delay time.Duration, failed bool) {
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
		qualifiedName := util.NewQualifiedName(obj.(pkgruntime.Object))
		s.deliver(qualifiedName, s.smallDelay, false)
	}
}

func (s *FederationSyncController) reconcile(qualifiedName util.QualifiedName) util.ReconciliationStatus {
	if !s.isSynced() {
		return util.StatusNotSynced
	}

	templateKind := s.typeConfig.GetTemplate().Kind
	key := qualifiedName.String()
	namespace := qualifiedName.Namespace

	targetKind := s.typeConfig.GetTarget().Kind
	if targetKind == util.NamespaceKind {
		namespace = qualifiedName.Name
		// TODO(font): Need a configurable or discoverable list of namespaces
		// to not propagate beyond just the default system namespaces e.g.
		// clusterregistry.
		if isSystemNamespace(s.fedNamespace, namespace) {
			return util.StatusAllOK
		}
	}

	glog.V(4).Infof("Starting to reconcile %v %v", templateKind, key)
	startTime := time.Now()
	defer glog.V(4).Infof("Finished reconciling %v %v (duration: %v)", templateKind, key, time.Now().Sub(startTime))

	template, err := s.objFromCache(s.templateStore, templateKind, key)
	if err != nil {
		return util.StatusError
	}
	if template == nil {
		return util.StatusAllOK
	}

	if template.GetDeletionTimestamp() != nil {
		err := s.delete(template, templateKind, qualifiedName)
		if err != nil {
			msg := "Failed to delete %s %q: %v"
			args := []interface{}{templateKind, qualifiedName, err}
			runtime.HandleError(fmt.Errorf(msg, args...))
			s.eventRecorder.Eventf(template, corev1.EventTypeWarning, "DeleteFailed", msg, args...)
			return util.StatusError
		}
		return util.StatusAllOK
	}

	glog.V(3).Infof("Ensuring finalizers exist on %s %q", templateKind, key)
	finalizedTemplate, err := s.deletionHelper.EnsureFinalizers(template)
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to ensure finalizers for %s %q: %v", templateKind, key, err))
		return util.StatusError
	}
	template = finalizedTemplate.(*unstructured.Unstructured)

	clusterNames, err := s.clusterNames()
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to get cluster list: %v", err))
		return util.StatusNotSynced
	}

	selectedClusters, unselectedClusters, err := s.placementPlugin.ComputePlacement(key, clusterNames)
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to compute placement for %s %q: %v", templateKind, key, err))
		return util.StatusError
	}

	var override *unstructured.Unstructured
	if s.overrideStore != nil {
		override, err = s.objFromCache(s.overrideStore, s.typeConfig.GetOverride().Kind, key)
		if err != nil {
			return util.StatusError
		}
	}

	propagatedVersionKey := util.QualifiedName{
		Namespace: namespace,
		Name:      s.versionName(qualifiedName.Name),
	}.String()
	// TODO(marun) LOCK!
	if s.pendingVersionUpdates.Has(propagatedVersionKey) {
		// TODO(marun) Need to revisit how namespace deletion affects
		// the version cache.  Ignoring may cause some unnecessary
		// updates, but that's better than looping endlessly.
		if targetKind != util.NamespaceKind {
			// A status update is pending
			return util.StatusNeedsRecheck
		}
	}
	propagatedVersionFromCache, err := s.rawObjFromCache(s.propagatedVersionStore,
		"PropagatedVersion", propagatedVersionKey)
	if err != nil {
		return util.StatusError
	}
	var propagatedVersion *fedv1a1.PropagatedVersion
	if propagatedVersionFromCache != nil {
		propagatedVersion = propagatedVersionFromCache.(*fedv1a1.PropagatedVersion)
	}

	return s.syncToClusters(selectedClusters, unselectedClusters, template, override, propagatedVersion)
}

func (s *FederationSyncController) rawObjFromCache(store cache.Store, kind, key string) (pkgruntime.Object, error) {
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

func (s *FederationSyncController) objFromCache(store cache.Store, kind, key string) (*unstructured.Unstructured, error) {
	obj, err := s.rawObjFromCache(store, kind, key)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, nil
	}
	return obj.(*unstructured.Unstructured), nil
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
func (s *FederationSyncController) delete(template pkgruntime.Object,
	kind string, qualifiedName util.QualifiedName) error {
	glog.V(3).Infof("Handling deletion of %s %q", kind, qualifiedName)

	_, err := s.deletionHelper.HandleObjectInUnderlyingClusters(template, kind)
	if err != nil {
		return err
	}

	if kind == util.NamespaceKind {
		// Return immediately if we are a namespace as it will be deleted
		// simply by removing its finalizers.
		return nil
	}

	versionName := s.versionName(qualifiedName.Name)
	err = s.fedClient.CoreV1alpha1().PropagatedVersions(qualifiedName.Namespace).Delete(versionName, nil)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	err = s.templateClient.Resources(qualifiedName.Namespace).Delete(qualifiedName.Name, nil)
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
	targetKind := s.typeConfig.GetTarget().Kind
	return common.PropagatedVersionName(targetKind, resourceName)
}

// syncToClusters ensures that the state of the given object is synchronized to
// member clusters.
func (s *FederationSyncController) syncToClusters(selectedClusters, unselectedClusters []string,
	template, override *unstructured.Unstructured, propagatedVersion *fedv1a1.PropagatedVersion) util.ReconciliationStatus {

	templateKind := s.typeConfig.GetTemplate().Kind
	key := util.NewQualifiedName(template).String()

	glog.V(3).Infof("Syncing %s %q in underlying clusters, selected clusters are: %s, unselected clusters are: %s",
		templateKind, key, selectedClusters, unselectedClusters)

	propagatedClusterVersions := getClusterVersions(template, override, propagatedVersion)

	operations, err := s.clusterOperations(selectedClusters, unselectedClusters, template,
		override, key, propagatedClusterVersions)
	if err != nil {
		s.eventRecorder.Eventf(template, corev1.EventTypeWarning, "FedClusterOperationsError",
			"Error obtaining sync operations for %s: %s error: %s", templateKind, key, err.Error())
		return util.StatusError
	}

	if len(operations) == 0 {
		return util.StatusAllOK
	}

	// TODO(marun) raise the visibility of operationErrors to aid in debugging
	updatedClusterVersions, operationErrors := s.updater.Update(operations)

	defer func() {
		err = updatePropagatedVersion(s.typeConfig, s.fedClient, updatedClusterVersions, template, override,
			propagatedVersion, selectedClusters, s.pendingVersionUpdates)
		if err != nil {
			runtime.HandleError(fmt.Errorf("Failed to record propagated version for %s %q: %v", templateKind,
				key, err))
			// Failure to record the propagated version does not imply
			// that propagation for the resource needs to be attempted
			// again.
		}
	}()

	if len(operationErrors) > 0 {
		runtime.HandleError(fmt.Errorf("Failed to execute updates for %s %q: %v", templateKind,
			key, operationErrors))
		return util.StatusError
	}

	return util.StatusAllOK
}

// getClusterVersions returns the cluster versions populated in the current
// propagated version object if it exists.
func getClusterVersions(template, override *unstructured.Unstructured, propagatedVersion *fedv1a1.PropagatedVersion) map[string]string {

	clusterVersions := make(map[string]string)

	if propagatedVersion == nil {
		return clusterVersions
	}

	templateVersion := template.GetResourceVersion()
	overrideVersion := ""
	if override != nil {
		overrideVersion = override.GetResourceVersion()
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
func updatePropagatedVersion(typeConfig typeconfig.Interface, fedClient fedclientset.Interface,
	updatedVersions map[string]string, template, override *unstructured.Unstructured,
	version *fedv1a1.PropagatedVersion, selectedClusters []string,
	pendingVersionUpdates sets.String) error {

	overrideVersion := ""
	if override != nil {
		overrideVersion = override.GetResourceVersion()
	}

	if version == nil {
		version := newVersion(updatedVersions, template, typeConfig.GetTarget().Kind, overrideVersion)
		createdVersion, err := fedClient.CoreV1alpha1().PropagatedVersions(version.Namespace).Create(version)
		if err != nil {
			return err
		}

		key := util.NewQualifiedName(createdVersion).String()
		// TODO(marun) add timeout to ensure against lost updates blocking propagation of a given resource
		pendingVersionUpdates.Insert(key)

		_, err = fedClient.CoreV1alpha1().PropagatedVersions(version.Namespace).UpdateStatus(createdVersion)
		if err != nil {
			pendingVersionUpdates.Delete(key)
		}
		return err
	}

	oldVersionStatus := version.Status
	templateVersion := template.GetResourceVersion()
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
			typeConfig.GetTemplate().Kind, util.NewQualifiedName(template).String())
		return nil
	}

	key := util.NewQualifiedName(version).String()
	pendingVersionUpdates.Insert(key)

	_, err := fedClient.CoreV1alpha1().PropagatedVersions(version.Namespace).UpdateStatus(version)
	if err != nil {
		pendingVersionUpdates.Delete(key)
	}
	return err
}

// newVersion initializes a new propagated version resource for the given
// cluster versions and template and override.
func newVersion(clusterVersions map[string]string, templateMeta metav1.Object, targetKind,
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
	if targetKind == util.NamespaceKind {
		namespace = templateMeta.GetName()
	} else {
		namespace = templateMeta.GetNamespace()
	}

	return &fedv1a1.PropagatedVersion{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      common.PropagatedVersionName(targetKind, templateMeta.GetName()),
		},
		Status: fedv1a1.PropagatedVersionStatus{
			TemplateVersion: templateMeta.GetResourceVersion(),
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
	template, override *unstructured.Unstructured, key string,
	clusterVersions map[string]string) ([]util.FederatedOperation, error) {

	operations := make([]util.FederatedOperation, 0)

	targetKind := s.typeConfig.GetTarget().Kind
	for _, clusterName := range selectedClusters {
		desiredObj, err := s.objectForCluster(template, override, clusterName)
		if err != nil {
			return nil, err
		}

		// TODO(marun) Wait until result of add operation has reached
		// the target store before attempting subsequent operations?
		// Otherwise the object won't be found but an add operation
		// will fail with AlreadyExists.
		clusterObj, found, err := s.informer.GetTargetStore().GetByKey(clusterName, key)
		if err != nil {
			wrappedErr := fmt.Errorf("Failed to get %s %q from cluster %q: %v", targetKind, key, clusterName, err)
			runtime.HandleError(wrappedErr)
			return nil, wrappedErr
		}

		var operationType util.FederatedOperationType = ""

		if found {
			clusterObj := clusterObj.(*unstructured.Unstructured)

			// If we're a namespace kind and this is an object for the primary
			// cluster, then skip the version comparison check as we do not
			// track the cluster version for namespaces in the primary cluster.
			// This avoids unnecessary updates that triggers an infinite loop
			// of continually adding finalizers and then removing finalizers,
			// causing PropagatedVersion to not keep up with the
			// ResourceVersions being updated.
			if targetKind == util.NamespaceKind {
				if util.IsPrimaryCluster(template, clusterObj) {
					continue
				}
			}

			desiredObj, err = s.objectForUpdateOp(desiredObj, clusterObj)
			if err != nil {
				wrappedErr := fmt.Errorf("Failed to determine desired object %s %q for cluster %q: %v", targetKind, key, clusterName, err)
				runtime.HandleError(wrappedErr)
				return nil, wrappedErr
			}

			version, ok := clusterVersions[clusterName]
			if !ok {
				// No target version recorded for template+override version
				operationType = util.OperationTypeUpdate
			} else {
				targetVersion := s.comparisonHelper.GetVersion(clusterObj)

				// Check if versions don't match. If they match then check its
				// ObjectMeta which only applies to resources where Generation
				// is used to track versions because Generation is only updated
				// when Spec changes.
				if version != targetVersion {
					operationType = util.OperationTypeUpdate
				} else if !s.comparisonHelper.Equivalent(desiredObj, clusterObj) {
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
			wrappedErr := fmt.Errorf("Failed to get %s %q from cluster %q: %v", targetKind, key, clusterName, err)
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

func (s *FederationSyncController) objectForCluster(template, override *unstructured.Unstructured, clusterName string) (*unstructured.Unstructured, error) {
	// Federation of namespaces uses Namespace resources as the
	// template for resource creation in member clusters. All other
	// federated types rely on a template type distinct from the
	// target type.
	//
	// Namespace is the only type that can contain other resources,
	// and adding a federation-specific container type would be
	// difficult or impossible. This implies that federation
	// primitives need to exist in regular namespaces.
	//
	// TODO(marun) Ensure this is reflected in documentation
	targetKind := s.typeConfig.GetTarget().Kind
	obj := &unstructured.Unstructured{}
	if targetKind == util.NamespaceKind {
		metadata, ok, err := unstructured.NestedMap(template.Object, "metadata")
		if err != nil {
			return nil, fmt.Errorf("Error retrieving namespace metadata", err)
		}
		if !ok {
			return nil, fmt.Errorf("Unable to retrieve namespace metadata")
		}
		// Retain only the target fields from the template
		targetFields := sets.NewString("name", "namespace", "labels", "annotations")
		for key, _ := range metadata {
			if !targetFields.Has(key) {
				delete(metadata, key)
			}
		}
		obj.Object = make(map[string]interface{})
		obj.Object["metadata"] = metadata
	} else {
		var ok bool
		var err error
		obj.Object, ok, err = unstructured.NestedMap(template.Object, "spec", "template")
		if err != nil {
			return nil, fmt.Errorf("Error retrieving template body", err)
		}
		if !ok {
			return nil, fmt.Errorf("Unable to retrieve template body")
		}
		// Avoid having to duplicate these details in the template or have
		// the name/namespace vary between the federation api and member
		// clusters.
		//
		// TODO(marun) this should be documented
		obj.SetName(template.GetName())
		obj.SetNamespace(template.GetNamespace())
		targetApiResource := s.typeConfig.GetTarget()
		obj.SetKind(targetApiResource.Kind)
		obj.SetAPIVersion(fmt.Sprintf("%s/%s", targetApiResource.Group, targetApiResource.Version))
	}

	if override == nil {
		return obj, nil
	}
	overrides, ok, err := unstructured.NestedSlice(override.Object, "spec", "overrides")
	if err != nil {
		return nil, fmt.Errorf("Error retrieving overrides for %q: %v", targetKind, err)
	}
	if !ok {
		return obj, nil
	}
	overridePath := s.typeConfig.GetOverridePath()
	if len(overridePath) == 0 {
		return nil, fmt.Errorf("Override path is missing for %q", targetKind)
	}
	overrideField := overridePath[len(overridePath)-1]
	for _, overrideInterface := range overrides {
		clusterOverride := overrideInterface.(map[string]interface{})
		if clusterOverride["clustername"] != clusterName {
			continue
		}
		data, ok := clusterOverride[overrideField]
		if ok {
			unstructured.SetNestedField(obj.Object, data, overridePath...)
		}
		break
	}

	return obj, nil
}

// TODO(marun) Support webhooks for custom update behavior
func (s *FederationSyncController) objectForUpdateOp(desiredObj, clusterObj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	// Pass the same ResourceVersion as in the cluster object for update operation, otherwise operation will fail.
	desiredObj.SetResourceVersion(clusterObj.GetResourceVersion())

	if s.typeConfig.GetTarget().Kind == util.ServiceKind {
		return serviceForUpdateOp(desiredObj, clusterObj)
	}
	return desiredObj, nil
}

func serviceForUpdateOp(desiredObj, clusterObj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	// ClusterIP and NodePort are allocated to Service by cluster, so retain the same if any while updating

	// Retain clusterip
	clusterIP, ok, err := unstructured.NestedString(clusterObj.Object, "spec", "clusterIP")
	if err != nil {
		return nil, fmt.Errorf("Error retrieving clusterIP from cluster service: %v", err)
	}
	// !ok could indicate that a cluster ip was not assigned
	if ok && clusterIP != "" {
		err := unstructured.SetNestedField(desiredObj.Object, clusterIP, "spec", "clusterIP")
		if err != nil {
			return nil, fmt.Errorf("Error setting clusterIP for service: %v", err)
		}
	}

	// Retain nodeports
	clusterPorts, ok, err := unstructured.NestedSlice(clusterObj.Object, "spec", "ports")
	if err != nil {
		return nil, fmt.Errorf("Error retrieving ports from cluster service: %v", err)
	}
	if !ok {
		return desiredObj, nil
	}
	var desiredPorts []interface{}
	desiredPorts, ok, err = unstructured.NestedSlice(desiredObj.Object, "spec", "ports")
	if err != nil {
		return nil, fmt.Errorf("Error retrieving ports from service: %v", err)
	}
	if !ok {
		desiredPorts = []interface{}{}
	}
	for desiredIndex, _ := range desiredPorts {
		for clusterIndex, _ := range clusterPorts {
			fPort := desiredPorts[desiredIndex].(map[string]interface{})
			cPort := clusterPorts[clusterIndex].(map[string]interface{})
			if !(fPort["name"] == cPort["name"] && fPort["protocol"] == cPort["protocol"] && fPort["port"] == cPort["port"]) {
				continue
			}
			nodePort, ok := cPort["nodePort"]
			if ok {
				cPort["nodePort"] = nodePort
			}
		}
	}
	err = unstructured.SetNestedSlice(desiredObj.Object, desiredPorts, "spec", "ports")
	if err != nil {
		return nil, fmt.Errorf("Error setting ports for service: %v", err)
	}

	return desiredObj, nil
}

// TODO (font): Externalize this list to a package var to allow it to be
// configurable.
func isSystemNamespace(fedNamespace, namespace string) bool {
	switch namespace {
	case "kube-system", fedNamespace:
		return true
	default:
		return false
	}
}
