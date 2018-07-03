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

package replicaschedulingpreference

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	fedschedulingv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/scheduling/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/replicaschedulingpreference/scheduler"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
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
	crclientset "k8s.io/cluster-registry/pkg/client/clientset/versioned"
)

const (
	allClustersKey = "ALL_CLUSTERS"
)

// ReplicaSchedulingPreferenceController syncronises the template, override
// and placement for a target deployment template with its spec (user preference).
type ReplicaSchedulingPreferenceController struct {
	// Used to allow time delay in triggering reconciliation
	// when any of RSP, target template, override or placement
	// changes.
	deliverer *util.DelayingDeliverer

	// For triggering reconciliation of all resources (only in
	// federation). This is used when a new cluster becomes available.
	clusterDeliverer *util.DelayingDeliverer

	// scheduler holds all the information and functionality
	// to handle the target objects of RSP
	scheduler *scheduler.Scheduler

	// Store for self (ReplicaSchedulingPreference)
	store cache.Store
	// Informer for self (ReplicaSchedulingPreference)
	controller cache.Controller

	// Work queue allowing parallel processing of resources
	workQueue workqueue.Interface

	// Backoff manager
	backoff *flowcontrol.Backoff

	// For events
	eventRecorder record.EventRecorder

	reviewDelay             time.Duration
	clusterAvailableDelay   time.Duration
	clusterUnavailableDelay time.Duration
	smallDelay              time.Duration
	updateTimeout           time.Duration
}

// ReplicaSchedulingPreferenceController starts a new controller for ReplicaSchedulingPreferences
func StartReplicaSchedulingPreferenceController(config *restclient.Config, stopChan <-chan struct{}, minimizeLatency bool) error {
	restclient.AddUserAgent(config, "replicaschedulingpreference-controller")
	fedClient := fedclientset.NewForConfigOrDie(config)
	kubeClient := kubeclientset.NewForConfigOrDie(config)
	crClient := crclientset.NewForConfigOrDie(config)
	controller, err := newReplicaSchedulingPreferenceController(fedClient, kubeClient, crClient)
	if err != nil {
		return err
	}
	if minimizeLatency {
		controller.minimizeLatency()
	}
	glog.Infof(fmt.Sprintf("Starting replicaschedulingpreferences controller"))
	controller.Run(stopChan)
	return nil
}

// newReplicaSchedulingPreferenceController returns a new ReplicaSchedulingPreference Controller for the given client
func newReplicaSchedulingPreferenceController(fedClient fedclientset.Interface, kubeClient kubeclientset.Interface, crClient crclientset.Interface) (*ReplicaSchedulingPreferenceController, error) {
	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	recorder := broadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: fmt.Sprintf("replicaschedulingpreference-controller")})

	s := &ReplicaSchedulingPreferenceController{
		reviewDelay:             time.Second * 10,
		clusterAvailableDelay:   time.Second * 20,
		clusterUnavailableDelay: time.Second * 60,
		smallDelay:              time.Second * 3,
		updateTimeout:           time.Second * 30,
		workQueue:               workqueue.New(),
		backoff:                 flowcontrol.NewBackOff(5*time.Second, time.Minute),
		eventRecorder:           recorder,
	}

	// Build delivereres for triggering reconciliations.
	s.deliverer = util.NewDelayingDeliverer()
	s.clusterDeliverer = util.NewDelayingDeliverer()

	s.store, s.controller = cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (pkgruntime.Object, error) {
				return fedClient.SchedulingV1alpha1().ReplicaSchedulingPreferences(metav1.NamespaceAll).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return fedClient.SchedulingV1alpha1().ReplicaSchedulingPreferences(metav1.NamespaceAll).Watch(options)
			},
		},
		&fedschedulingv1a1.ReplicaSchedulingPreference{},
		util.NoResyncPeriod,
		util.NewTriggerOnAllChanges(
			func(obj pkgruntime.Object) {
				s.deliverObj(obj, 0, false)
			}),
	)

	s.scheduler = scheduler.NewReplicaScheduler(
		fedClient,
		kubeClient,
		crClient,
		func(obj pkgruntime.Object) {
			s.deliverObj(obj, 0, false)
		},
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
		})

	return s, nil
}

// minimizeLatency reduces delays and timeouts to make the controller more responsive (useful for testing).
func (s *ReplicaSchedulingPreferenceController) minimizeLatency() {
	s.clusterAvailableDelay = time.Second
	s.clusterUnavailableDelay = time.Second
	s.reviewDelay = 50 * time.Millisecond
	s.smallDelay = 20 * time.Millisecond
	s.updateTimeout = 5 * time.Second
}

func (s *ReplicaSchedulingPreferenceController) Run(stopChan <-chan struct{}) {
	go s.controller.Run(stopChan)
	s.scheduler.Start(stopChan)

	s.deliverer.StartWithHandler(func(item *util.DelayingDelivererItem) {
		s.workQueue.Add(item)
	})
	s.clusterDeliverer.StartWithHandler(func(_ *util.DelayingDelivererItem) {
		s.reconcileOnClusterChange()
	})

	go wait.Until(s.worker, time.Second, stopChan)

	util.StartBackoffGC(s.backoff, stopChan)

	// Ensure all goroutines are cleaned up when the stop channel closes
	go func() {
		<-stopChan
		s.workQueue.ShutDown()
		s.deliverer.Stop()
		s.clusterDeliverer.Stop()
		s.scheduler.Stop()
	}()
}

func (s *ReplicaSchedulingPreferenceController) worker() {
	for {
		item, quit := s.workQueue.Get()
		if quit {
			return
		}
		typedItem := item.(*util.DelayingDelivererItem)
		qualifiedName := typedItem.Value.(*util.QualifiedName)
		status := s.reconcile(*qualifiedName)
		s.workQueue.Done(typedItem)

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

func (s *ReplicaSchedulingPreferenceController) deliverObj(obj pkgruntime.Object, delay time.Duration, failed bool) {
	qualifiedName := util.NewQualifiedName(obj)
	s.deliver(qualifiedName, delay, failed)
}

// Adds backoff to delay if this delivery is related to some failure. Resets backoff if there was no failure.
func (s *ReplicaSchedulingPreferenceController) deliver(qualifiedName util.QualifiedName, delay time.Duration, failed bool) {
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
func (s *ReplicaSchedulingPreferenceController) isSynced() bool {
	return s.controller.HasSynced() && s.scheduler.HasSynced()
}

// The function triggers reconciliation of all known RSP resources.
func (s *ReplicaSchedulingPreferenceController) reconcileOnClusterChange() {
	if !s.isSynced() {
		s.clusterDeliverer.DeliverAt(allClustersKey, nil, time.Now().Add(s.clusterAvailableDelay))
	}
	for _, obj := range s.store.List() {
		qualifiedName := util.NewQualifiedName(obj.(pkgruntime.Object))
		s.deliver(qualifiedName, s.smallDelay, false)
	}
}

func (s *ReplicaSchedulingPreferenceController) reconcile(qualifiedName util.QualifiedName) util.ReconciliationStatus {
	if !s.isSynced() {
		return util.StatusNotSynced
	}

	kind := "ReplicaSchedulingPreference"
	key := qualifiedName.String()

	glog.V(4).Infof("Starting to reconcile %s controller triggerred key named %v", kind, key)
	startTime := time.Now()
	defer glog.V(4).Infof("Finished reconciling %s controller triggerred key named %v (duration: %v)", kind, key, time.Now().Sub(startTime))

	rsp, err := s.objFromCache(s.store, kind, key)
	if err != nil {
		return util.StatusAllOK
	}
	if rsp == nil {
		// Nothing to do
		return util.StatusAllOK
	}

	typedRsp, ok := rsp.(*fedschedulingv1a1.ReplicaSchedulingPreference)
	if !ok {
		runtime.HandleError(fmt.Errorf("Incorrect runtime object for RSP: %v", rsp))
		return util.StatusError
	}

	return s.scheduler.Reconcile(typedRsp, qualifiedName)
}

func (s *ReplicaSchedulingPreferenceController) objFromCache(store cache.Store, kind, key string) (pkgruntime.Object, error) {
	cachedObj, exist, err := store.GetByKey(key)
	if err != nil {
		wrappedErr := fmt.Errorf("Failed to query store while reconciling RSP controller, triggerred by %s named %q: %v", kind, key, err)
		runtime.HandleError(wrappedErr)
		return nil, err
	}
	if !exist {
		return nil, nil
	}
	return cachedObj.(pkgruntime.Object).DeepCopyObject(), nil
}
