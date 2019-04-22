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
	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	kubeclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	genericclient "github.com/kubernetes-sigs/federation-v2/pkg/client/generic"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/sync/dispatch"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
)

const (
	allClustersKey = "ALL_CLUSTERS"
)

// FederationSyncController synchronizes the state of a federated type
// to clusters that are members of the federation.
type FederationSyncController struct {
	// TODO(marun) add comment
	worker util.ReconcileWorker

	// For triggering reconciliation of all target resources. This is
	// used when a new cluster becomes available.
	clusterDeliverer *util.DelayingDeliverer

	// Contains resources present in members of federation.
	informer util.FederatedInformer
	// For updating members of federation.
	updater util.FederatedUpdater

	// For events
	eventRecorder record.EventRecorder

	clusterAvailableDelay   time.Duration
	clusterUnavailableDelay time.Duration
	smallDelay              time.Duration
	updateTimeout           time.Duration

	typeConfig typeconfig.Interface

	fedAccessor FederatedResourceAccessor
}

// StartFederationSyncController starts a new sync controller for a type config
func StartFederationSyncController(controllerConfig *util.ControllerConfig, stopChan <-chan struct{}, typeConfig typeconfig.Interface, fedNamespaceAPIResource *metav1.APIResource) error {
	controller, err := newFederationSyncController(controllerConfig, typeConfig, fedNamespaceAPIResource)
	if err != nil {
		return err
	}
	if controllerConfig.MinimizeLatency {
		controller.minimizeLatency()
	}
	glog.Infof(fmt.Sprintf("Starting sync controller for %q", typeConfig.GetFederatedType().Kind))
	controller.Run(stopChan)
	return nil
}

// newFederationSyncController returns a new sync controller for the configuration
func newFederationSyncController(controllerConfig *util.ControllerConfig, typeConfig typeconfig.Interface, fedNamespaceAPIResource *metav1.APIResource) (*FederationSyncController, error) {
	federatedTypeAPIResource := typeConfig.GetFederatedType()
	userAgent := fmt.Sprintf("%s-controller", strings.ToLower(federatedTypeAPIResource.Kind))

	// Initialize non-dynamic clients first to avoid polluting config
	client := genericclient.NewForConfigOrDieWithUserAgent(controllerConfig.KubeConfig, userAgent)
	kubeClient := kubeclient.NewForConfigOrDie(controllerConfig.KubeConfig)

	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	recorder := broadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: userAgent})

	s := &FederationSyncController{
		clusterAvailableDelay:   controllerConfig.ClusterAvailableDelay,
		clusterUnavailableDelay: controllerConfig.ClusterUnavailableDelay,
		smallDelay:              time.Second * 3,
		updateTimeout:           time.Second * 30,
		eventRecorder:           recorder,
		typeConfig:              typeConfig,
	}

	s.worker = util.NewReconcileWorker(s.reconcile, util.WorkerTiming{
		ClusterSyncDelay: s.clusterAvailableDelay,
	})

	// Build deliverer for triggering cluster reconciliations.
	s.clusterDeliverer = util.NewDelayingDeliverer()

	targetAPIResource := typeConfig.GetTarget()

	// Federated informer on the resource type in members of federation.
	var err error
	s.informer, err = util.NewFederatedInformer(
		controllerConfig,
		client,
		&targetAPIResource,
		func(obj pkgruntime.Object) {
			qualifiedName := util.NewQualifiedName(obj)
			s.worker.EnqueueForRetry(qualifiedName)
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
	if err != nil {
		return nil, err
	}

	// Federated updater along with Create/Update/Delete operations.
	s.updater = util.NewFederatedUpdater(s.informer, targetAPIResource.Kind, s.updateTimeout, s.eventRecorder,
		func(client util.ResourceClient, rawObj pkgruntime.Object) (string, error) {
			obj := rawObj.(*unstructured.Unstructured)
			createdObj, err := client.Resources(obj.GetNamespace()).Create(obj, metav1.CreateOptions{})
			if err != nil {
				return "", err
			}
			return util.ObjectVersion(createdObj), err
		},
		func(client util.ResourceClient, rawObj pkgruntime.Object) (string, error) {
			obj := rawObj.(*unstructured.Unstructured)
			updatedObj, err := client.Resources(obj.GetNamespace()).Update(obj, metav1.UpdateOptions{})
			if err != nil {
				return "", err
			}
			return util.ObjectVersion(updatedObj), err
		},
		func(client util.ResourceClient, obj pkgruntime.Object) (string, error) {
			qualifiedName := util.NewQualifiedName(obj)
			orphanDependents := false
			return "", client.Resources(qualifiedName.Namespace).Delete(qualifiedName.Name, &metav1.DeleteOptions{OrphanDependents: &orphanDependents})
		})

	s.fedAccessor, err = NewFederatedResourceAccessor(
		controllerConfig, typeConfig, fedNamespaceAPIResource,
		client, s.worker.EnqueueObject, s.informer, s.updater, recorder)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// minimizeLatency reduces delays and timeouts to make the controller more responsive (useful for testing).
func (s *FederationSyncController) minimizeLatency() {
	s.clusterAvailableDelay = time.Second
	s.clusterUnavailableDelay = time.Second
	s.smallDelay = 20 * time.Millisecond
	s.updateTimeout = 5 * time.Second
	s.worker.SetDelay(50*time.Millisecond, s.clusterAvailableDelay)
}

func (s *FederationSyncController) Run(stopChan <-chan struct{}) {
	s.fedAccessor.Run(stopChan)
	s.informer.Start()
	s.clusterDeliverer.StartWithHandler(func(_ *util.DelayingDelivererItem) {
		s.reconcileOnClusterChange()
	})

	s.worker.Run(stopChan)

	// Ensure all goroutines are cleaned up when the stop channel closes
	go func() {
		<-stopChan
		s.informer.Stop()
		s.clusterDeliverer.Stop()
	}()
}

// Check whether all data stores are in sync. False is returned if any of the informer/stores is not yet
// synced with the corresponding api server.
func (s *FederationSyncController) isSynced() bool {
	if !s.informer.ClustersSynced() {
		glog.V(2).Infof("Cluster list not synced")
		return false
	}
	if !s.fedAccessor.HasSynced() {
		// The fed accessor will have logged why sync is not yet
		// complete.
		return false
	}

	// TODO(marun) set clusters as ready in the test fixture?
	clusters, err := s.informer.GetReadyClusters()
	if err != nil {
		runtime.HandleError(errors.Wrap(err, "Failed to get ready clusters"))
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
	s.fedAccessor.VisitFederatedResources(func(obj interface{}) {
		qualifiedName := util.NewQualifiedName(obj.(pkgruntime.Object))
		s.worker.EnqueueWithDelay(qualifiedName, s.smallDelay)
	})
}

func (s *FederationSyncController) reconcile(qualifiedName util.QualifiedName) util.ReconciliationStatus {
	if !s.isSynced() {
		return util.StatusNotSynced
	}

	kind := s.typeConfig.GetFederatedType().Kind

	// TODO(marun) Handle the case where the resource has the managed
	// label but does not have a managing resource.  Strip the label
	// or remove the resource?

	fedResource, err := s.fedAccessor.FederatedResource(qualifiedName)
	if err != nil {
		runtime.HandleError(errors.Wrapf(err, "Error creating FederatedResource helper for %s %q", kind, qualifiedName))
		return util.StatusError
	}
	if fedResource == nil {
		return util.StatusAllOK
	}

	key := fedResource.FederatedName().String()

	glog.V(4).Infof("Starting to reconcile %s %q", kind, key)
	startTime := time.Now()
	defer glog.V(4).Infof("Finished reconciling %s %q (duration: %v)", kind, key, time.Since(startTime))

	if fedResource.MarkedForDeletion() {
		glog.V(3).Infof("Handling deletion of %s %q", kind, key)
		err := fedResource.EnsureDeletion()
		if err != nil {
			// TODO(marun) Log a warning rather than handle an error
			// when waiting for resources to be deleted in underlying
			// clusters.

			// It is not possible to record events on resources marked for deletion.
			runtime.HandleError(errors.Wrapf(err, "Unable to delete %s %q", kind, key))
			return util.StatusError
		}
		// It should now be possible to garbage collect the finalization target.
		return util.StatusAllOK
	}
	glog.V(3).Infof("Ensuring finalizer exists on %s %q", kind, key)
	err = fedResource.EnsureFinalizer()
	if err != nil {
		fedResource.RecordError("EnsureFinalizerError", errors.Wrap(err, "Failed to ensure finalizer"))
		return util.StatusError
	}

	clusters, err := s.informer.GetReadyClusters()
	if err != nil {
		fedResource.RecordError("ClusterRetrievalError", errors.Wrap(err, "Failed to get cluster list"))
		return util.StatusNotSynced
	}

	return s.syncToClusters(fedResource, clusters)
}

// syncToClusters ensures that the state of the given object is
// synchronized to the provided clusters.
func (s *FederationSyncController) syncToClusters(fedResource FederatedResource, clusters []*fedv1a1.FederatedCluster) util.ReconciliationStatus {
	selectedClusterNames, err := fedResource.ComputePlacement(clusters)
	if err != nil {
		fedResource.RecordError("ComputePlacementError", errors.Wrap(err, "Failed to compute placement"))
		return util.StatusError
	}

	kind := fedResource.TargetKind()
	key := fedResource.TargetName().String()
	glog.V(4).Infof("Syncing %s %q in underlying clusters, selected clusters are: %s", kind, key, selectedClusterNames)

	dispatcher := dispatch.NewManagedDispatcher(s.informer.GetClientForCluster, fedResource)

	status := util.StatusAllOK
	for _, cluster := range clusters {
		clusterName := cluster.Name

		rawClusterObj, _, err := s.informer.GetTargetStore().GetByKey(clusterName, key)
		if err != nil {
			fedResource.RecordError("TargetRetrievalError",
				errors.Wrapf(err, "Failed to retrieve cluster object for cluster %q", clusterName))
			// Ensure an error status is returned but continue
			// processing updates for other clusters.
			status = util.StatusError
			continue
		}

		var clusterObj *unstructured.Unstructured
		if rawClusterObj != nil {
			clusterObj = rawClusterObj.(*unstructured.Unstructured)
		}

		// Resource should not exist in the named cluster
		if !selectedClusterNames.Has(clusterName) {
			if clusterObj == nil || clusterObj.GetDeletionTimestamp() != nil {
				continue
			}
			if fedResource.IsNamespaceInHostCluster(clusterObj) {
				// Host cluster namespace needs to have the managed
				// label removed so it won't be cached anymore.
				dispatcher.RemoveManagedLabel(clusterName, clusterObj)
			} else {
				dispatcher.Delete(clusterName)
			}
			continue
		}

		// Resource should appear in the named cluster

		// TODO(marun) Consider waiting until the result of resource
		// creation has reached the target store before attempting
		// subsequent operations.  Otherwise the object won't be found
		// but an add operation will fail with AlreadyExists.
		if clusterObj == nil {
			dispatcher.Create(clusterName)
		} else {
			dispatcher.Update(clusterName, clusterObj)
		}
	}
	ok, err := dispatcher.Wait()
	if err != nil {
		fedResource.RecordError("OperationTimeoutError", err)
		status = util.StatusError
	}
	if !ok {
		status = util.StatusError
	}
	updatedVersionMap := dispatcher.VersionMap()
	// Always attempt to update versions even if the updater reported errors.
	err = fedResource.UpdateVersions(selectedClusterNames.List(), updatedVersionMap)
	if err != nil {
		// Versioning of federated resources is an optimization to
		// avoid unnecessary updates, and failure to record version
		// information does not indicate a failure of propagation.
		runtime.HandleError(err)
	}

	return status
}
