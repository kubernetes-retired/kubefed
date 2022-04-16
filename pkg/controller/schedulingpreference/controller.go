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
	"strings"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	fedv1b1 "sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	"sigs.k8s.io/kubefed/pkg/controller/util"
	"sigs.k8s.io/kubefed/pkg/metrics"
	"sigs.k8s.io/kubefed/pkg/schedulingtypes"
)

const (
	allClustersKey = "ALL_CLUSTERS"
)

// SchedulingPreferenceController synchronises the template, override
// and placement for a target template with its spec (user preference).
type SchedulingPreferenceController struct {
	// For triggering reconciliation of all scheduling resources. This
	// is used when a new cluster becomes available.
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
	klog.Infof("Starting replicaschedulingpreferences controller")
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
	recorder := broadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "replicaschedulingpreference-controller"})

	s := &SchedulingPreferenceController{
		clusterAvailableDelay:   config.ClusterAvailableDelay,
		clusterUnavailableDelay: config.ClusterUnavailableDelay,
		smallDelay:              time.Second * 3,
		eventRecorder:           recorder,
	}

	s.worker = util.NewReconcileWorker(strings.ToLower(schedulingType.Kind), s.reconcile, util.WorkerOptions{
		WorkerTiming: util.WorkerTiming{
			ClusterSyncDelay: s.clusterAvailableDelay,
		},
	})

	eventHandlers := schedulingtypes.SchedulerEventHandlers{
		KubeFedEventHandler: s.worker.EnqueueObject,
		ClusterEventHandler: func(obj runtimeclient.Object) {
			qualifiedName := util.NewQualifiedName(obj)
			s.worker.EnqueueForRetry(qualifiedName)
		},
		ClusterLifecycleHandlers: &util.ClusterLifecycleHandlerFuncs{
			ClusterAvailable: func(cluster *fedv1b1.KubeFedCluster) {
				// When new cluster becomes available process all the target resources again.
				s.clusterDeliverer.DeliverAt(allClustersKey, nil, time.Now().Add(s.clusterAvailableDelay))
			},
			// When a cluster becomes unavailable process all the target resources again.
			ClusterUnavailable: func(cluster *fedv1b1.KubeFedCluster, _ []interface{}) {
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
		kubeConfig,
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
		qualifiedName := util.NewQualifiedName(obj.(runtimeclient.Object))
		s.worker.EnqueueWithDelay(qualifiedName, s.smallDelay)
	}
}

func (s *SchedulingPreferenceController) reconcile(qualifiedName util.QualifiedName) util.ReconciliationStatus {
	defer metrics.UpdateControllerReconcileDurationFromStart("schedulingpreferencecontroller", time.Now())

	if !s.isSynced() {
		return util.StatusNotSynced
	}

	kind := s.scheduler.SchedulingKind()
	key := qualifiedName.String()

	klog.V(4).Infof("Starting to reconcile %s controller triggered key named %v", kind, key)
	startTime := time.Now()
	defer func() {
		klog.V(4).Infof("Finished reconciling %s controller triggered key named %v (duration: %v)", kind, key, time.Since(startTime))
	}()

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

func (s *SchedulingPreferenceController) objFromCache(store cache.Store, kind, key string) (runtimeclient.Object, error) {
	cachedObj, exist, err := store.GetByKey(key)
	if err != nil {
		wrappedErr := errors.Wrapf(err, "Failed to query store while reconciling RSP controller, triggered by %s named %q", kind, key)
		runtime.HandleError(wrappedErr)
		return nil, err
	}
	if !exist {
		return nil, nil
	}
	return cachedObj.(runtimeclient.Object).DeepCopyObject().(runtimeclient.Object), nil
}
