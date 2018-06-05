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
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/golang/glog"
	fedschedulingv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/federatedscheduling/v1alpha1"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/federation/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset_generated/clientset"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util/planner"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util/podanalyzer"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
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
	crclientset "k8s.io/cluster-registry/pkg/client/clientset_generated/clientset"
)

const (
	allClustersKey = "ALL_CLUSTERS"
)

// ReplicaSchedulingPreferenceController syncronises the template, override
// and placement for a target deployment template with its spec (user preference).
// TODO: make this usable atleast for current known scheduling types (deployments and replicasets)
type ReplicaSchedulingPreferenceController struct {
	// Used to allow time delay in triggering reconciliation
	// when any of RSP, target template, override or placement
	// changes.
	deliverer *util.DelayingDeliverer

	// For triggering reconciliation of all resources (only in
	// federation). This is used when a new cluster becomes available.
	clusterDeliverer *util.DelayingDeliverer

	// Contains target resources present in members of federation.
	targetInformer util.FederatedInformer

	// Informs about pods present in members of federation.
	// TODO: Rather then a separate informer, a typed client
	// interface can be added into the targetInformer.
	podInformer util.FederatedInformer

	// Store for self (ReplicaSchedulingPreference)
	store cache.Store
	// Informer for self (ReplicaSchedulingPreference)
	controller cache.Controller
	// Client to the federation API
	fedClient fedclientset.Interface

	// Store for the templates of the federated type
	templateStore cache.Store
	// Informer for the templates of the federated type
	templateController cache.Controller

	// Store for the override directives of the federated type
	overrideStore cache.Store
	// Informer controller for override directives of the federated type
	overrideController cache.Controller

	// Store for the placements of the federated type
	placementStore cache.Store
	// Informer controller for placements of the federated type
	placementController cache.Controller

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
func StartReplicaSchedulingPreferenceController(fedConfig, kubeConfig, crConfig *restclient.Config, stopChan <-chan struct{}, minimizeLatency bool) error {
	userAgent := "replicaschedulingpreference-controller"
	restclient.AddUserAgent(fedConfig, userAgent)
	fedClient := fedclientset.NewForConfigOrDie(fedConfig)
	restclient.AddUserAgent(kubeConfig, userAgent)
	kubeClient := kubeclientset.NewForConfigOrDie(kubeConfig)
	restclient.AddUserAgent(crConfig, userAgent)
	crClient := crclientset.NewForConfigOrDie(crConfig)
	controller, err := newReplicaSchedulingPreferenceController(fedConfig, fedClient, kubeClient, crClient)
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
func newReplicaSchedulingPreferenceController(fedConfig *restclient.Config, fedClient fedclientset.Interface, kubeClient kubeclientset.Interface, crClient crclientset.Interface) (*ReplicaSchedulingPreferenceController, error) {
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
		fedClient:               fedClient,
	}

	// Build delivereres for triggering reconciliations.
	s.deliverer = util.NewDelayingDeliverer()
	s.clusterDeliverer = util.NewDelayingDeliverer()

	// Start informers on the resources for the federated type
	deliverObj := func(obj pkgruntime.Object) {
		s.deliverObj(obj, 0, false)
	}

	s.store, s.controller = cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (pkgruntime.Object, error) {
				return fedClient.FederatedschedulingV1alpha1().ReplicaSchedulingPreferences(metav1.NamespaceAll).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return fedClient.FederatedschedulingV1alpha1().ReplicaSchedulingPreferences(metav1.NamespaceAll).Watch(options)
			},
		},
		&fedschedulingv1a1.ReplicaSchedulingPreference{},
		util.NoResyncPeriod,
		util.NewTriggerOnAllChanges(deliverObj),
	)

	s.templateStore, s.templateController = cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (pkgruntime.Object, error) {
				return fedClient.FederationV1alpha1().FederatedDeployments(metav1.NamespaceAll).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return fedClient.FederationV1alpha1().FederatedDeployments(metav1.NamespaceAll).Watch(options)
			},
		},
		&fedv1a1.FederatedDeployment{},
		util.NoResyncPeriod,
		util.NewTriggerOnAllChanges(deliverObj),
	)

	s.overrideStore, s.overrideController = cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (pkgruntime.Object, error) {
				return fedClient.FederationV1alpha1().FederatedDeploymentOverrides(metav1.NamespaceAll).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return fedClient.FederationV1alpha1().FederatedDeploymentOverrides(metav1.NamespaceAll).Watch(options)
			},
		},
		&fedv1a1.FederatedDeploymentOverride{},
		util.NoResyncPeriod,
		util.NewTriggerOnAllChanges(deliverObj),
	)

	s.placementStore, s.placementController = cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (pkgruntime.Object, error) {
				return fedClient.FederationV1alpha1().FederatedDeploymentPlacements(metav1.NamespaceAll).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return fedClient.FederationV1alpha1().FederatedDeploymentPlacements(metav1.NamespaceAll).Watch(options)
			},
		},
		&fedv1a1.FederatedDeploymentPlacement{},
		util.NoResyncPeriod,
		util.NewTriggerOnAllChanges(deliverObj),
	)

	// Federated informer on the resource type in members of federation.
	s.targetInformer = util.NewFederatedInformer(
		fedClient,
		kubeClient,
		crClient,
		&metav1.APIResource{
			Name:       "deployments",
			Group:      "apps",
			Version:    "v1",
			Kind:       "deployment",
			Namespaced: true,
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
		},
	)

	// Federated informer watching pods in members of federation.
	s.podInformer = util.NewFederatedInformer(
		fedClient,
		kubeClient,
		crClient,
		&metav1.APIResource{
			Name:       "pods",
			Group:      "",
			Version:    "v1",
			Kind:       "pod",
			Namespaced: true,
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
		},
	)

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
	go s.templateController.Run(stopChan)
	go s.overrideController.Run(stopChan)
	go s.placementController.Run(stopChan)

	s.targetInformer.Start()
	s.podInformer.Start()
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
		s.targetInformer.Stop()
		s.podInformer.Stop()
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

func (s *ReplicaSchedulingPreferenceController) worker() {
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

func (s *ReplicaSchedulingPreferenceController) deliverObj(obj pkgruntime.Object, delay time.Duration, failed bool) {
	// TODO: right now this works on the RSP being same "ns/name" as the
	// deploymenttemplate. Update this to rsp being able to use
	// targetRef (kind+name) with multiple kinds.
	// For reviewers - Plan is to use Objectmeta.OwnerReferences by setting it to
	// created rsp into the target object. This way when reconcile is called
	// the information will be available to link objects both ways.
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
	if !s.targetInformer.ClustersSynced() {
		glog.V(2).Infof("Cluster list not synced")
		return false
	}
	clusters, err := s.targetInformer.GetReadyClusters()
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to get ready clusters: %v", err))
		return false
	}
	if !s.targetInformer.GetTargetStore().ClustersSynced(clusters) {
		return false
	}

	if !s.podInformer.ClustersSynced() {
		glog.V(2).Infof("Cluster list not synced")
		return false
	}
	clusters, err = s.podInformer.GetReadyClusters()
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to get ready clusters: %v", err))
		return false
	}
	if !s.podInformer.GetTargetStore().ClustersSynced(clusters) {
		return false
	}

	if !s.controller.HasSynced() ||
		!s.templateController.HasSynced() ||
		!s.overrideController.HasSynced() ||
		!s.placementController.HasSynced() {
		return false
	}

	return true
}

// The function triggers reconciliation of all target federated resources.
func (s *ReplicaSchedulingPreferenceController) reconcileOnClusterChange() {
	if !s.isSynced() {
		s.clusterDeliverer.DeliverAt(allClustersKey, nil, time.Now().Add(s.clusterAvailableDelay))
	}
	for _, obj := range s.templateStore.List() {
		qualifiedName := util.NewQualifiedName(obj.(pkgruntime.Object))
		s.deliver(qualifiedName, s.smallDelay, false)
	}
}

func (s *ReplicaSchedulingPreferenceController) reconcile(qualifiedName util.QualifiedName) reconciliationStatus {
	if !s.isSynced() {
		return statusNotSynced
	}

	kind := "ReplicaSchedulingPreference"
	key := qualifiedName.String()

	glog.V(4).Infof("Starting to reconcile %v %v", kind, key)
	startTime := time.Now()
	defer glog.V(4).Infof("Finished reconciling %v %v (duration: %v)", kind, key, time.Now().Sub(startTime))

	rsp, err := s.objFromCache(s.store, kind, key)
	if err != nil {
		return statusError
	}
	if rsp == nil {
		return statusAllOK
	}

	typedRsp, ok := rsp.(*fedschedulingv1a1.ReplicaSchedulingPreference)
	if !ok {
		runtime.HandleError(fmt.Errorf("Incorrect runtime object for RSP: %v", rsp))
		return statusNotSynced
	}

	clusterNames, err := s.clusterNames()
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to get cluster list: %v", err))
		return statusNotSynced
	}
	if len(clusterNames) == 0 {
		// no joined clusters, nothing to do
		return statusAllOK
	}

	template, err := s.objFromCache(s.templateStore, "FederatedDeployment", key)
	if err != nil {
		return statusError
	}
	if template == nil {
		return statusAllOK
	}

	typedTemplate := template.(*fedv1a1.FederatedDeployment)
	result, err := s.GetSchedulingResult(typedRsp, &typedTemplate.Spec.Template, key, clusterNames)
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to compute the schedule information for RSP %q: %v", key, err))
		return statusError
	}

	updateFederationTargets(s.fedClient, qualifiedName, result)
	return statusAllOK
}

func (s *ReplicaSchedulingPreferenceController) objFromCache(store cache.Store, kind, key string) (pkgruntime.Object, error) {
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

func (s *ReplicaSchedulingPreferenceController) clusterNames() ([]string, error) {
	clusters, err := s.targetInformer.GetReadyClusters()
	if err != nil {
		return nil, err
	}
	clusterNames := []string{}
	for _, cluster := range clusters {
		clusterNames = append(clusterNames, cluster.Name)
	}

	return clusterNames, nil
}

func (s *ReplicaSchedulingPreferenceController) GetSchedulingResult(fedPref *fedschedulingv1a1.ReplicaSchedulingPreference, obj pkgruntime.Object, key string, clusterNames []string) (map[string]int64, error) {
	objectGetter := func(clusterName, key string) (interface{}, bool, error) {
		return s.targetInformer.GetTargetStore().GetByKey(clusterName, key)
	}
	podsGetter := func(clusterName string, unstructuredObj *unstructured.Unstructured) (pkgruntime.Object, error) {
		client, err := s.podInformer.GetClientForCluster(clusterName)
		if err != nil {
			return nil, err
		}
		selectorLabels, ok := unstructured.NestedStringMap(unstructuredObj.Object, "spec", "selector", "matchLabels")
		if !ok {
			return nil, fmt.Errorf("missing selector on object: %v", err)
		}

		label := labels.SelectorFromSet(labels.Set(selectorLabels))
		if err != nil {
			return nil, fmt.Errorf("invalid selector: %v", err)
		}
		unstructuredPodList, err := client.Resources(unstructuredObj.GetNamespace()).List(metav1.ListOptions{LabelSelector: label.String()})
		if err != nil || unstructuredPodList == nil {
			return nil, err
		}
		return unstructuredPodList, nil
	}

	currentReplicasPerCluster, estimatedCapacity, err := clustersReplicaState(clusterNames, key, objectGetter, podsGetter)
	if err != nil {
		return nil, err
	}

	// TODO: Move this to API defaulting logic
	if len(fedPref.Spec.Clusters) == 0 {
		fedPref.Spec.Clusters = map[string]fedschedulingv1a1.ClusterPreferences{
			"*": {Weight: 1},
		}
	}

	plnr := planner.NewPlanner(fedPref)
	return schedule(plnr, key, clusterNames, currentReplicasPerCluster, estimatedCapacity), nil
}

func schedule(planner *planner.Planner, key string, clusterNames []string, currentReplicasPerCluster map[string]int64, estimatedCapacity map[string]int64) map[string]int64 {

	scheduleResult, overflow := planner.Plan(clusterNames, currentReplicasPerCluster, estimatedCapacity, key)

	// TODO: Check if we really need to place the template in clusters
	// with 0 replicas. Override replicas would be set to 0 in this case.
	result := make(map[string]int64)
	for clusterName := range currentReplicasPerCluster {
		result[clusterName] = 0
	}

	for clusterName, replicas := range scheduleResult {
		result[clusterName] = replicas
	}
	for clusterName, replicas := range overflow {
		result[clusterName] += replicas
	}

	if glog.V(4) {
		buf := bytes.NewBufferString(fmt.Sprintf("Schedule - %q\n", key))
		sort.Strings(clusterNames)
		for _, clusterName := range clusterNames {
			cur := currentReplicasPerCluster[clusterName]
			target := scheduleResult[clusterName]
			fmt.Fprintf(buf, "%s: current: %d target: %d", clusterName, cur, target)
			if over, found := overflow[clusterName]; found {
				fmt.Fprintf(buf, " overflow: %d", over)
			}
			if capacity, found := estimatedCapacity[clusterName]; found {
				fmt.Fprintf(buf, " capacity: %d", capacity)
			}
			fmt.Fprintf(buf, "\n")
		}
		glog.V(4).Infof(buf.String())
	}
	return result
}

// clusterReplicaState returns information about the scheduling state of the pods running in the federated clusters.
func clustersReplicaState(
	clusterNames []string,
	key string,
	objectGetter func(clusterName string, key string) (interface{}, bool, error),
	podsGetter func(clusterName string, obj *unstructured.Unstructured) (pkgruntime.Object, error)) (currentReplicasPerCluster map[string]int64, estimatedCapacity map[string]int64, err error) {

	currentReplicasPerCluster = make(map[string]int64)
	estimatedCapacity = make(map[string]int64)

	for _, clusterName := range clusterNames {
		obj, exists, err := objectGetter(clusterName, key)
		if err != nil {
			return nil, nil, err
		}
		if !exists {
			continue
		}

		unstructuredObj := obj.(*unstructured.Unstructured)
		replicas, ok := unstructured.NestedInt64(unstructuredObj.Object, "spec", "replicas")
		if !ok {
			replicas = int64(0)
		}
		readyReplicas, ok := unstructured.NestedInt64(unstructuredObj.Object, "status", "readyreplicas")
		if !ok {
			readyReplicas = int64(0)
		}

		if replicas == readyReplicas {
			currentReplicasPerCluster[clusterName] = readyReplicas
		} else {
			currentReplicasPerCluster[clusterName] = int64(0)
			pods, err := podsGetter(clusterName, unstructuredObj)
			if err != nil {
				return nil, nil, err
			}

			//TODO: Update AnalysePods to use typed podList.
			// Unstructured list seems not very suitable for functions like
			// AnalysePods() and there are many possibilities of unchecked
			// errors, if extensive set or get of fields is done on unstructured
			// object. A good mechanism might be to get a typed client
			// in FedInformer which is much easier to work with in PodLists.
			podList := pods.(*unstructured.UnstructuredList)
			podStatus := podanalyzer.AnalyzePods(podList, time.Now())
			currentReplicasPerCluster[clusterName] = int64(podStatus.RunningAndReady) // include pending as well?
			unschedulable := int64(podStatus.Unschedulable)
			if unschedulable > 0 {
				estimatedCapacity[clusterName] = replicas - unschedulable
			}
		}
	}
	return currentReplicasPerCluster, estimatedCapacity, nil
}

func updateFederationTargets(fedClient fedclientset.Interface, qualifiedName util.QualifiedName, result map[string]int64) error {
	newClusterNames := []string{}
	for name := range result {
		newClusterNames = append(newClusterNames, name)
	}

	err := updatePlacement(fedClient, qualifiedName, newClusterNames)
	if err != nil {
		return err
	}

	err = updateOverrides(fedClient, qualifiedName, result)
	if err != nil {
		return err
	}

	return nil
}

func updatePlacement(fedClient fedclientset.Interface, qualifiedName util.QualifiedName, newClusterNames []string) error {
	placement, err := fedClient.FederationV1alpha1().FederatedDeploymentPlacements(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		newPlacement := &fedv1a1.FederatedDeploymentPlacement{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: qualifiedName.Namespace,
				Name:      qualifiedName.Name,
			},
			Spec: fedv1a1.FederatedDeploymentPlacementSpec{
				ClusterNames: newClusterNames,
			},
		}
		_, err := fedClient.FederationV1alpha1().FederatedDeploymentPlacements(qualifiedName.Namespace).Create(newPlacement)
		return err
	}

	if placementUpdateNeeded(placement.Spec.ClusterNames, newClusterNames) {
		// TODO: do we need to make a copy of this object
		newPlacement := placement
		newPlacement.Spec.ClusterNames = newClusterNames
		_, err := fedClient.FederationV1alpha1().FederatedDeploymentPlacements(qualifiedName.Namespace).Update(newPlacement)
		if err != nil {
			return err
		}
	}

	return nil
}

func updateOverrides(fedClient fedclientset.Interface, qualifiedName util.QualifiedName, result map[string]int64) error {
	override, err := fedClient.FederationV1alpha1().FederatedDeploymentOverrides(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		newOverride := &fedv1a1.FederatedDeploymentOverride{
			ObjectMeta: metav1.ObjectMeta{
				Name:      qualifiedName.Name,
				Namespace: qualifiedName.Namespace,
			},
			Spec: fedv1a1.FederatedDeploymentOverrideSpec{},
		}

		for clusterName, replicas := range result {
			var r int32 = int32(replicas)
			clusterOverride := fedv1a1.FederatedDeploymentClusterOverride{
				ClusterName: clusterName,
				Replicas:    &r,
			}
			newOverride.Spec.Overrides = append(newOverride.Spec.Overrides, clusterOverride)
		}

		_, err := fedClient.FederationV1alpha1().FederatedDeploymentOverrides(qualifiedName.Namespace).Create(newOverride)
		return err
	}

	if overrideUpdateNeeded(override.Spec, result) {
		// TODO: do we need to make a copy of this object
		newOverride := override
		newOverride.Spec = fedv1a1.FederatedDeploymentOverrideSpec{}
		for clusterName, replicas := range result {
			var r int32 = int32(replicas)
			clusterOverride := fedv1a1.FederatedDeploymentClusterOverride{
				ClusterName: clusterName,
				Replicas:    &r,
			}
			newOverride.Spec.Overrides = append(newOverride.Spec.Overrides, clusterOverride)
		}
		_, err := fedClient.FederationV1alpha1().FederatedDeploymentOverrides(qualifiedName.Namespace).Update(newOverride)
		if err != nil {
			return err
		}
	}

	return nil
}

// These assume that there would be no duplicate clusternames
func placementUpdateNeeded(names, newNames []string) bool {
	sort.Strings(names)
	sort.Strings(newNames)
	return !reflect.DeepEqual(names, newNames)
}

func overrideUpdateNeeded(overrideSpec fedv1a1.FederatedDeploymentOverrideSpec, result map[string]int64) bool {
	resultLen := len(result)
	checkLen := 0
	for _, override := range overrideSpec.Overrides {
		replicas, ok := result[override.ClusterName]
		if !ok || (override.Replicas == nil) || (int32(replicas) != *override.Replicas) {
			return true
		}
		checkLen += 1
	}

	return checkLen != resultLen
}
