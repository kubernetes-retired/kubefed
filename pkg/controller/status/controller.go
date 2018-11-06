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

	"github.com/golang/glog"
	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
)

const (
	allClustersKey = "ALL_CLUSTERS"
)

// FederationStatusController collects the status of a federated type
// from clusters that are members of the federation.
type FederationStatusController struct {
	// For triggering reconciliation of all target resources. This is
	// used when a new cluster becomes available.
	clusterDeliverer *util.DelayingDeliverer

	// Contains resources present in members of federation.
	informer util.FederatedInformer

	// Store for the templates of the federated type
	templateStore cache.Store
	// Informer for the templates of the federated type
	templateController cache.Controller

	// Store for the status of the federated type
	statusStore cache.Store
	// Informer for the status of the federated type
	statusController cache.Controller

	worker util.ReconcileWorker

	clusterAvailableDelay   time.Duration
	clusterUnavailableDelay time.Duration
	smallDelay              time.Duration

	typeConfig typeconfig.Interface

	fedClient    fedclientset.Interface
	statusClient util.ResourceClient

	fedNamespace string
}

// StartFederationStatusController starts a new status controller for a type config
func StartFederationStatusController(controllerConfig *util.ControllerConfig, stopChan <-chan struct{}, typeConfig typeconfig.Interface) error {
	controller, err := newFederationStatusController(controllerConfig, typeConfig)
	if err != nil {
		return err
	}
	if controllerConfig.MinimizeLatency {
		controller.minimizeLatency()
	}
	glog.Infof(fmt.Sprintf("Starting status controller for %q", typeConfig.GetTemplate().Kind))
	controller.Run(stopChan)
	return nil
}

// newFederationStatusController returns a new status controller for the federated type
func newFederationStatusController(controllerConfig *util.ControllerConfig, typeConfig typeconfig.Interface) (*FederationStatusController, error) {
	templateAPIResource := typeConfig.GetTemplate()
	statusAPIResource := typeConfig.GetStatus()
	userAgent := fmt.Sprintf("%s-controller", strings.ToLower(statusAPIResource.Kind))

	// Initialize non-dynamic clients first to avoid polluting config
	fedClient, kubeClient, crClient := controllerConfig.AllClients(userAgent)

	pool := dynamic.NewDynamicClientPool(controllerConfig.KubeConfig)

	templateClient, err := util.NewResourceClient(pool, &templateAPIResource)
	if err != nil {
		return nil, err
	}

	statusClient, err := util.NewResourceClient(pool, statusAPIResource)
	if err != nil {
		return nil, err
	}

	s := &FederationStatusController{
		clusterAvailableDelay:   controllerConfig.ClusterAvailableDelay,
		clusterUnavailableDelay: controllerConfig.ClusterUnavailableDelay,
		smallDelay:              time.Second * 3,
		typeConfig:              typeConfig,
		fedClient:               fedClient,
		statusClient:            statusClient,
		fedNamespace:            controllerConfig.FederationNamespace,
	}

	s.worker = util.NewReconcileWorker(s.reconcile, util.WorkerTiming{
		ClusterSyncDelay: s.clusterAvailableDelay,
	})

	// Build deliverer for triggering cluster reconciliations.
	s.clusterDeliverer = util.NewDelayingDeliverer()

	// Start informers on the resources for the federated type
	enqueueObj := s.worker.EnqueueObject

	targetNamespace := controllerConfig.TargetNamespace

	s.templateStore, s.templateController = util.NewResourceInformer(templateClient, targetNamespace, enqueueObj)
	s.statusStore, s.statusController = util.NewResourceInformer(statusClient, targetNamespace, enqueueObj)

	targetAPIResource := typeConfig.GetTarget()

	// Federated informer on the resource type in members of federation.
	s.informer = util.NewFederatedInformer(
		fedClient,
		kubeClient,
		crClient,
		controllerConfig.FederationNamespaces,
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

	return s, nil
}

// minimizeLatency reduces delays and timeouts to make the controller more responsive (useful for testing).
func (s *FederationStatusController) minimizeLatency() {
	s.clusterAvailableDelay = time.Second
	s.clusterUnavailableDelay = time.Second
	s.smallDelay = 20 * time.Millisecond
	s.worker.SetDelay(50*time.Millisecond, s.clusterAvailableDelay)
}

// Run runs the status controller
func (s *FederationStatusController) Run(stopChan <-chan struct{}) {
	go s.templateController.Run(stopChan)
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
func (s *FederationStatusController) isSynced() bool {
	if !s.informer.ClustersSynced() {
		glog.V(2).Infof("Cluster list not synced")
		return false
	}
	if !s.templateController.HasSynced() {
		glog.V(2).Infof("Template not synced")
		return false
	}
	if !s.statusController.HasSynced() {
		glog.V(2).Infof("Status not synced")
		return false
	}

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
func (s *FederationStatusController) reconcileOnClusterChange() {
	if !s.isSynced() {
		s.clusterDeliverer.DeliverAt(allClustersKey, nil, time.Now().Add(s.clusterAvailableDelay))
	}
	for _, obj := range s.templateStore.List() {
		qualifiedName := util.NewQualifiedName(obj.(pkgruntime.Object))
		s.worker.EnqueueWithDelay(qualifiedName, s.smallDelay)
	}
}

func (s *FederationStatusController) reconcile(qualifiedName util.QualifiedName) util.ReconciliationStatus {
	if !s.isSynced() {
		return util.StatusNotSynced
	}

	templateKind := s.typeConfig.GetTemplate().Kind
	statusKind := s.typeConfig.GetStatus().Kind
	key := qualifiedName.String()

	glog.V(4).Infof("Starting to reconcile %v %v", statusKind, key)
	startTime := time.Now()
	defer glog.V(4).Infof("Finished reconciling %v %v (duration: %v)", statusKind, key, time.Now().Sub(startTime))

	template, err := s.objFromCache(s.templateStore, templateKind, key)
	if err != nil {
		return util.StatusError
	}

	if template == nil || template.GetDeletionTimestamp() != nil {
		glog.V(4).Infof("No template for %v %v found", templateKind, key)
		// Status object is removed by GC. So we don't have to do anything more here.
		return util.StatusAllOK
	}

	clusterNames, err := s.clusterNames()
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to get cluster list: %v", err))
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

	resourceGroupVersion := schema.GroupVersion{Group: s.typeConfig.GetStatus().Group, Version: s.typeConfig.GetStatus().Version}
	federatedResource := util.FederatedResource{
		TypeMeta: metav1.TypeMeta{
			Kind:       s.typeConfig.GetStatus().Kind,
			APIVersion: resourceGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      qualifiedName.Name,
			Namespace: qualifiedName.Namespace,
			// Add ownership of status object to corresponding template object,
			// so that status object is deleted when template is deleted.
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: template.GetAPIVersion(),
				Kind:       template.GetKind(),
				Name:       template.GetName(),
				UID:        template.GetUID(),
			}},
		},
		ClusterStatus: clusterStatus,
	}
	status, err := util.GetUnstructured(federatedResource)
	if err != nil {
		glog.Errorf("Failed to convert to Unstructured: %s %q: %v", statusKind, key, err)
		return util.StatusError
	}

	if existingStatus == nil {
		_, err = s.statusClient.Resources(qualifiedName.Namespace).Create(status)
		if err != nil {
			runtime.HandleError(fmt.Errorf("Failed to create status object for federated type %s %q: %v", statusKind, key, err))
			return util.StatusNeedsRecheck
		}
	} else if !reflect.DeepEqual(existingStatus.Object["clusterStatus"], status.Object["clusterStatus"]) {
		existingStatus.Object["clusterStatus"] = status.Object["clusterStatus"]
		_, err = s.statusClient.Resources(qualifiedName.Namespace).Update(existingStatus)
		if err != nil {
			runtime.HandleError(fmt.Errorf("Failed to update status object for federated type %s %q: %v", statusKind, key, err))
			return util.StatusNeedsRecheck
		}
	}

	return util.StatusAllOK
}

func (s *FederationStatusController) rawObjFromCache(store cache.Store, kind, key string) (pkgruntime.Object, error) {
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

func (s *FederationStatusController) objFromCache(store cache.Store, kind, key string) (*unstructured.Unstructured, error) {
	obj, err := s.rawObjFromCache(store, kind, key)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, nil
	}
	return obj.(*unstructured.Unstructured), nil
}

func (s *FederationStatusController) clusterNames() ([]string, error) {
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
func (s *FederationStatusController) clusterStatuses(clusterNames []string, key string) ([]util.ResourceClusterStatus, error) {
	clusterStatus := []util.ResourceClusterStatus{}

	targetKind := s.typeConfig.GetTarget().Kind
	for _, clusterName := range clusterNames {
		clusterObj, exist, err := s.informer.GetTargetStore().GetByKey(clusterName, key)
		if err != nil {
			wrappedErr := fmt.Errorf("Failed to get %s %q from cluster %q: %v", targetKind, key, clusterName, err)
			runtime.HandleError(wrappedErr)
			return nil, wrappedErr
		}

		var status map[string]interface{}
		if exist {
			clusterObj := clusterObj.(*unstructured.Unstructured)

			var found bool
			status, found, err = unstructured.NestedMap(clusterObj.Object, "status")
			if err != nil || !found {
				wrappedErr := fmt.Errorf("Failed to get status of cluster resource object %s %q for cluster %q: %v", targetKind, key, clusterName, err)
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
