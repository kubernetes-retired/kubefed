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

package status

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"

	"sigs.k8s.io/kubefed/pkg/apis/core/typeconfig"
	fedv1b1 "sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	genericclient "sigs.k8s.io/kubefed/pkg/client/generic"
	"sigs.k8s.io/kubefed/pkg/controller/util"
)

const (
	allClustersKey = "ALL_CLUSTERS"
)

// KubeFedStatusController collects the status of resources in member
// clusters.
type KubeFedStatusController struct {
	// For triggering reconciliation of all target resources. This is
	// used when a new cluster becomes available.
	clusterDeliverer *util.DelayingDeliverer

	// Informer for resources in member clusters
	informer util.FederatedInformer

	// Store for the federated type
	federatedStore cache.Store
	// Informer for the federated type
	federatedController cache.Controller

	// Store for the status of the federated type
	statusStore cache.Store
	// Informer for the status of the federated type
	statusController cache.Controller

	worker util.ReconcileWorker

	clusterAvailableDelay   time.Duration
	clusterUnavailableDelay time.Duration
	smallDelay              time.Duration

	typeConfig typeconfig.Interface

	client       genericclient.Client
	statusClient util.ResourceClient

	fedNamespace string
}

// StartKubeFedStatusController starts a new status controller for a type config
func StartKubeFedStatusController(controllerConfig *util.ControllerConfig, stopChan <-chan struct{}, typeConfig typeconfig.Interface) error {
	controller, err := newKubeFedStatusController(controllerConfig, typeConfig)
	if err != nil {
		return err
	}
	if controllerConfig.MinimizeLatency {
		controller.minimizeLatency()
	}
	klog.Infof(fmt.Sprintf("Starting status controller for %q", typeConfig.GetFederatedType().Kind))
	controller.Run(stopChan)
	return nil
}

// newKubeFedStatusController returns a new status controller for the federated type
func newKubeFedStatusController(controllerConfig *util.ControllerConfig, typeConfig typeconfig.Interface) (*KubeFedStatusController, error) {
	federatedAPIResource := typeConfig.GetFederatedType()
	statusAPIResource := typeConfig.GetStatusType()
	if statusAPIResource == nil {
		return nil, errors.Errorf("Status collection is not supported for %q", federatedAPIResource.Kind)
	}
	userAgent := fmt.Sprintf("%s-controller", strings.ToLower(statusAPIResource.Kind))
	client := genericclient.NewForConfigOrDieWithUserAgent(controllerConfig.KubeConfig, userAgent)

	federatedTypeClient, err := util.NewResourceClient(controllerConfig.KubeConfig, &federatedAPIResource)
	if err != nil {
		return nil, err
	}

	statusClient, err := util.NewResourceClient(controllerConfig.KubeConfig, statusAPIResource)
	if err != nil {
		return nil, err
	}

	s := &KubeFedStatusController{
		clusterAvailableDelay:   controllerConfig.ClusterAvailableDelay,
		clusterUnavailableDelay: controllerConfig.ClusterUnavailableDelay,
		smallDelay:              time.Second * 3,
		typeConfig:              typeConfig,
		client:                  client,
		statusClient:            statusClient,
		fedNamespace:            controllerConfig.KubeFedNamespace,
	}

	s.worker = util.NewReconcileWorker(s.reconcile, util.WorkerTiming{
		ClusterSyncDelay: s.clusterAvailableDelay,
	})

	// Build deliverer for triggering cluster reconciliations.
	s.clusterDeliverer = util.NewDelayingDeliverer()

	// Start informers on the resources for the federated type
	enqueueObj := s.worker.EnqueueObject

	targetNamespace := controllerConfig.TargetNamespace

	targetAPIResource := typeConfig.GetTargetType()
	s.federatedStore, s.federatedController = util.NewResourceInformer(federatedTypeClient, targetNamespace, &targetAPIResource, enqueueObj)
	s.statusStore, s.statusController = util.NewResourceInformer(statusClient, targetNamespace, statusAPIResource, enqueueObj)

	// Federated informer for resources in member clusters
	s.informer, err = util.NewFederatedInformer(
		controllerConfig,
		client,
		&targetAPIResource,
		func(obj pkgruntime.Object) {
			qualifiedName := util.NewQualifiedName(obj)
			s.worker.EnqueueForRetry(qualifiedName)
		},
		&util.ClusterLifecycleHandlerFuncs{
			ClusterAvailable: func(cluster *fedv1b1.KubeFedCluster) {
				// When new cluster becomes available process all the target resources again.
				s.clusterDeliverer.DeliverAt(allClustersKey, nil, time.Now().Add(s.clusterAvailableDelay))
			},
			// When a cluster becomes unavailable process all the target resources again.
			ClusterUnavailable: func(cluster *fedv1b1.KubeFedCluster, _ []interface{}) {
				s.clusterDeliverer.DeliverAt(allClustersKey, nil, time.Now().Add(s.clusterUnavailableDelay))
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// minimizeLatency reduces delays and timeouts to make the controller more responsive (useful for testing).
func (s *KubeFedStatusController) minimizeLatency() {
	s.clusterAvailableDelay = time.Second
	s.clusterUnavailableDelay = time.Second
	s.smallDelay = 20 * time.Millisecond
	s.worker.SetDelay(50*time.Millisecond, s.clusterAvailableDelay)
}

// Run runs the status controller
func (s *KubeFedStatusController) Run(stopChan <-chan struct{}) {
	go s.federatedController.Run(stopChan)
	go s.statusController.Run(stopChan)
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
func (s *KubeFedStatusController) isSynced() bool {
	if !s.informer.ClustersSynced() {
		klog.V(2).Infof("Cluster list not synced")
		return false
	}
	if !s.federatedController.HasSynced() {
		klog.V(2).Infof("Federated type not synced")
		return false
	}
	if !s.statusController.HasSynced() {
		klog.V(2).Infof("Status not synced")
		return false
	}

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
func (s *KubeFedStatusController) reconcileOnClusterChange() {
	if !s.isSynced() {
		s.clusterDeliverer.DeliverAt(allClustersKey, nil, time.Now().Add(s.clusterAvailableDelay))
	}
	for _, obj := range s.federatedStore.List() {
		qualifiedName := util.NewQualifiedName(obj.(pkgruntime.Object))
		s.worker.EnqueueWithDelay(qualifiedName, s.smallDelay)
	}
}

func (s *KubeFedStatusController) reconcile(qualifiedName util.QualifiedName) util.ReconciliationStatus {
	if !s.isSynced() {
		return util.StatusNotSynced
	}

	federatedKind := s.typeConfig.GetFederatedType().Kind
	statusKind := s.typeConfig.GetStatusType().Kind
	key := qualifiedName.String()

	klog.V(4).Infof("Starting to reconcile %v %v", statusKind, key)
	startTime := time.Now()
	defer func() {
		klog.V(4).Infof("Finished reconciling %v %v (duration: %v)", statusKind, key, time.Since(startTime))
	}()

	fedObject, err := s.objFromCache(s.federatedStore, federatedKind, key)
	if err != nil {
		return util.StatusError
	}

	if fedObject == nil || fedObject.GetDeletionTimestamp() != nil {
		klog.V(4).Infof("No federated type for %v %v found", federatedKind, key)
		// Status object is removed by GC. So we don't have to do anything more here.
		return util.StatusAllOK
	}

	clusterNames, err := s.clusterNames()
	if err != nil {
		runtime.HandleError(errors.Wrap(err, "Failed to get cluster list"))
		return util.StatusNotSynced
	}

	clusterStatus, err := s.clusterStatuses(clusterNames, key)
	if err != nil {
		return util.StatusError
	}

	existingStatus, err := s.objFromCache(s.statusStore, statusKind, key)
	if err != nil {
		return util.StatusError
	}

	resourceGroupVersion := schema.GroupVersion{Group: s.typeConfig.GetStatusType().Group, Version: s.typeConfig.GetStatusType().Version}
	federatedResource := util.FederatedResource{
		TypeMeta: metav1.TypeMeta{
			Kind:       s.typeConfig.GetStatusType().Kind,
			APIVersion: resourceGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      qualifiedName.Name,
			Namespace: qualifiedName.Namespace,
			// Add ownership of status object to corresponding
			// federated object, so that status object is deleted when
			// the federated object is deleted.
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: fedObject.GetAPIVersion(),
				Kind:       fedObject.GetKind(),
				Name:       fedObject.GetName(),
				UID:        fedObject.GetUID(),
			}},
		},
		ClusterStatus: clusterStatus,
	}
	status, err := util.GetUnstructured(federatedResource)
	if err != nil {
		klog.Errorf("Failed to convert to Unstructured: %s %q: %v", statusKind, key, err)
		return util.StatusError
	}

	if existingStatus == nil {
		_, err = s.statusClient.Resources(qualifiedName.Namespace).Create(status, metav1.CreateOptions{})
		if err != nil {
			runtime.HandleError(errors.Wrapf(err, "Failed to create status object for federated type %s %q", statusKind, key))
			return util.StatusNeedsRecheck
		}
	} else if !reflect.DeepEqual(existingStatus.Object["clusterStatus"], status.Object["clusterStatus"]) {
		if status.Object["clusterStatus"] == nil {
			status.Object["clusterStatus"] = make([]util.ResourceClusterStatus, 0)
		}
		existingStatus.Object["clusterStatus"] = status.Object["clusterStatus"]
		_, err = s.statusClient.Resources(qualifiedName.Namespace).Update(existingStatus, metav1.UpdateOptions{})
		if err != nil {
			runtime.HandleError(errors.Wrapf(err, "Failed to update status object for federated type %s %q", statusKind, key))
			return util.StatusNeedsRecheck
		}
	}

	return util.StatusAllOK
}

func (s *KubeFedStatusController) rawObjFromCache(store cache.Store, kind, key string) (pkgruntime.Object, error) {
	cachedObj, exist, err := store.GetByKey(key)
	if err != nil {
		wrappedErr := errors.Wrapf(err, "Failed to query %s store for %q", kind, key)
		runtime.HandleError(wrappedErr)
		return nil, err
	}
	if !exist {
		return nil, nil
	}
	return cachedObj.(pkgruntime.Object).DeepCopyObject(), nil
}

func (s *KubeFedStatusController) objFromCache(store cache.Store, kind, key string) (*unstructured.Unstructured, error) {
	obj, err := s.rawObjFromCache(store, kind, key)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, nil
	}
	return obj.(*unstructured.Unstructured), nil
}

func (s *KubeFedStatusController) clusterNames() ([]string, error) {
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

// clusterStatuses returns the resource status in member cluster.
func (s *KubeFedStatusController) clusterStatuses(clusterNames []string, key string) ([]util.ResourceClusterStatus, error) {
	clusterStatus := []util.ResourceClusterStatus{}

	targetKind := s.typeConfig.GetTargetType().Kind
	for _, clusterName := range clusterNames {
		clusterObj, exist, err := s.informer.GetTargetStore().GetByKey(clusterName, key)
		if err != nil {
			wrappedErr := errors.Wrapf(err, "Failed to get %s %q from cluster %q", targetKind, key, clusterName)
			runtime.HandleError(wrappedErr)
			return nil, wrappedErr
		}

		var status map[string]interface{}
		if exist {
			clusterObj := clusterObj.(*unstructured.Unstructured)

			var found bool
			status, found, err = unstructured.NestedMap(clusterObj.Object, "status")
			if err != nil || !found {
				wrappedErr := errors.Wrapf(err, "Failed to get status of cluster resource object %s %q for cluster %q", targetKind, key, clusterName)
				runtime.HandleError(wrappedErr)
			}
		}
		resourceClusterStatus := util.ResourceClusterStatus{ClusterName: clusterName, Status: status}
		clusterStatus = append(clusterStatus, resourceClusterStatus)
	}

	sort.Slice(clusterStatus, func(i, j int) bool {
		return clusterStatus[i].ClusterName < clusterStatus[j].ClusterName
	})
	return clusterStatus, nil
}
