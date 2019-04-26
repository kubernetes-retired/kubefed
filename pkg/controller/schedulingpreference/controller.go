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

package schedulingpreference

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/schedulingtypes"
	corev1 "k8s.io/api/core/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

const (
	allClustersKey = "ALL_CLUSTERS"
)

// SchedulingPreferenceController synchronises the template, override
// and placement for a target template with its spec (user preference).
type SchedulingPreferenceController struct {
	// For triggering reconciliation of all resources (only in
	// federation). This is used when a new cluster becomes available.
	clusterDeliverer *util.DelayingDeliverer

	// scheduler holds all the information and functionality
	// to handle the target objects of given type
	scheduler schedulingtypes.Scheduler

	// Store for self
	store cache.Store
	// Informer for self
	controller cache.Controller

	worker util.ReconcileWorker

	// For events
	eventRecorder record.EventRecorder

	clusterAvailableDelay   time.Duration
	clusterUnavailableDelay time.Duration
	smallDelay              time.Duration
	updateTimeout           time.Duration
}

// SchedulingPreferenceController starts a new controller for given type of SchedulingPreferences
func StartSchedulingPreferenceController(config *util.ControllerConfig, schedulingType schedulingtypes.SchedulingType, stopChannel <-chan struct{}) (schedulingtypes.Scheduler, error) {
	controller, err := newSchedulingPreferenceController(config, schedulingType)
	if err != nil {
		return nil, err
	}
	if config.MinimizeLatency {
		controller.minimizeLatency()
	}
	glog.Infof(fmt.Sprintf("Starting replicaschedulingpreferences controller"))
	controller.Run(stopChannel)
	return controller.scheduler, nil
}

// newSchedulingPreferenceController returns a new SchedulingPreference Controller for the given type
func newSchedulingPreferenceController(config *util.ControllerConfig, schedulingType schedulingtypes.SchedulingType) (*SchedulingPreferenceController, error) {
	userAgent := fmt.Sprintf("%s-controller", schedulingType.Kind)
	kubeConfig := restclient.CopyConfig(config.KubeConfig)
	restclient.AddUserAgent(kubeConfig, userAgent)
	kubeClient, err := kubeclientset.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	recorder := broadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: fmt.Sprintf("replicaschedulingpreference-controller")})

	s := &SchedulingPreferenceController{
		clusterAvailableDelay:   config.ClusterAvailableDelay,
		clusterUnavailableDelay: config.ClusterUnavailableDelay,
		smallDelay:              time.Second * 3,
		updateTimeout:           time.Second * 30,
		eventRecorder:           recorder,
	}

	s.worker = util.NewReconcileWorker(s.reconcile, util.WorkerTiming{
		ClusterSyncDelay: s.clusterAvailableDelay,
	})

	eventHandlers := schedulingtypes.SchedulerEventHandlers{
		FederationEventHandler: s.worker.EnqueueObject,
		ClusterEventHandler: func(obj pkgruntime.Object) {
			qualifiedName := util.NewQualifiedName(obj)
			s.worker.EnqueueForRetry(qualifiedName)
		},
		ClusterLifecycleHandlers: &util.ClusterLifecycleHandlerFuncs{
			ClusterAvailable: func(cluster *fedv1a1.FederatedCluster) {
				// When new cluster becomes available process all the target resources again.
				s.clusterDeliverer.DeliverAt(allClustersKey, nil, time.Now().Add(s.clusterAvailableDelay))
			},
			// When a cluster becomes unavailable process all the target resources again.
			ClusterUnavailable: func(cluster *fedv1a1.FederatedCluster, _ []interface{}) {
				s.clusterDeliverer.DeliverAt(allClustersKey, nil, time.Now().Add(s.clusterUnavailableDelay))
			},
		},
	}
	scheduler, err := schedulingType.SchedulerFactory(config, eventHandlers)
	if err != nil {
		return nil, err
	}
	s.scheduler = scheduler

	// Build deliverer for triggering cluster reconciliations.
	s.clusterDeliverer = util.NewDelayingDeliverer()

	s.store, s.controller, err = util.NewGenericInformer(
		config.KubeConfig,
		config.TargetNamespace,
		s.scheduler.ObjectType(),
		util.NoResyncPeriod,
		s.worker.EnqueueObject,
	)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// minimizeLatency reduces delays and timeouts to make the controller more responsive (useful for testing).
func (s *SchedulingPreferenceController) minimizeLatency() {
	s.clusterAvailableDelay = time.Second
	s.clusterUnavailableDelay = time.Second
	s.smallDelay = 20 * time.Millisecond
	s.updateTimeout = 5 * time.Second
	s.worker.SetDelay(50*time.Millisecond, s.clusterAvailableDelay)
}

func (s *SchedulingPreferenceController) Run(stopChan <-chan struct{}) {
	go s.controller.Run(stopChan)
	s.scheduler.Start()

	s.clusterDeliverer.StartWithHandler(func(_ *util.DelayingDelivererItem) {
		s.reconcileOnClusterChange()
	})

	s.worker.Run(stopChan)

	// Ensure all goroutines are cleaned up when the stop channel closes
	go func() {
		<-stopChan
		s.clusterDeliverer.Stop()
		s.scheduler.Stop()
	}()
}

// Check whether all data stores are in sync. False is returned if any of the informer/stores is not yet
// synced with the corresponding api server.
func (s *SchedulingPreferenceController) isSynced() bool {
	return s.controller.HasSynced() && s.scheduler.HasSynced()
}

// The function triggers reconciliation of all known RSP resources.
func (s *SchedulingPreferenceController) reconcileOnClusterChange() {
	if !s.isSynced() {
		s.clusterDeliverer.DeliverAt(allClustersKey, nil, time.Now().Add(s.clusterAvailableDelay))
	}
	for _, obj := range s.store.List() {
		qualifiedName := util.NewQualifiedName(obj.(pkgruntime.Object))
		s.worker.EnqueueWithDelay(qualifiedName, s.smallDelay)
	}
}

func (s *SchedulingPreferenceController) reconcile(qualifiedName util.QualifiedName) util.ReconciliationStatus {
	if !s.isSynced() {
		return util.StatusNotSynced
	}

	kind := s.scheduler.SchedulingKind()
	key := qualifiedName.String()

	glog.V(4).Infof("Starting to reconcile %s controller triggered key named %v", kind, key)
	startTime := time.Now()
	defer glog.V(4).Infof("Finished reconciling %s controller triggered key named %v (duration: %v)", kind, key, time.Since(startTime))

	obj, err := s.objFromCache(s.store, kind, key)
	if err != nil {
		return util.StatusAllOK
	}
	if obj == nil {
		// Nothing to do
		return util.StatusAllOK
	}

	return s.scheduler.Reconcile(obj, qualifiedName)
}

func (s *SchedulingPreferenceController) objFromCache(store cache.Store, kind, key string) (pkgruntime.Object, error) {
	cachedObj, exist, err := store.GetByKey(key)
	if err != nil {
		wrappedErr := errors.Wrapf(err, "Failed to query store while reconciling RSP controller, triggered by %s named %q", kind, key)
		runtime.HandleError(wrappedErr)
		return nil, err
	}
	if !exist {
		return nil, nil
	}
	return cachedObj.(pkgruntime.Object).DeepCopyObject(), nil
}
