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

package schedulingtypes

import (
	"bytes"
	"fmt"
	"sort"
	"time"

	fedschedulingv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/scheduling/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	. "github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util/planner"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util/podanalyzer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	kubeclientset "k8s.io/client-go/kubernetes"
	crclientset "k8s.io/cluster-registry/pkg/client/clientset/versioned"

	"github.com/golang/glog"
	"github.com/kubernetes-sigs/federation-v2/pkg/schedulingtypes/adapters"
)

const (
	RSPKind = "ReplicaSchedulingPreference"
)

func init() {
	RegisterSchedulingType(RSPKind, NewReplicaScheduler)
}

type pluginArgs struct {
	FederationNamespaces
	kubeClient             kubeclientset.Interface
	crClient               crclientset.Interface
	federationEventHandler func(pkgruntime.Object)
	clusterEventHandler    func(pkgruntime.Object)
	handlers               *ClusterLifecycleHandlerFuncs
}

type ReplicaScheduler struct {
	plugins map[string]*Plugin
	pluginArgs

	fedClient       fedclientset.Interface
	podInformer     FederatedInformer
	targetNamespace string
}

func NewReplicaScheduler(fedClient fedclientset.Interface, kubeClient kubeclientset.Interface, crClient crclientset.Interface, namespaces FederationNamespaces, federationEventHandler, clusterEventHandler func(pkgruntime.Object), handlers *ClusterLifecycleHandlerFuncs) Scheduler {
	scheduler := &ReplicaScheduler{
		plugins: make(map[string]*Plugin),
		pluginArgs: pluginArgs{
			kubeClient:             kubeClient,
			crClient:               crClient,
			FederationNamespaces:   namespaces,
			federationEventHandler: federationEventHandler,
			clusterEventHandler:    clusterEventHandler,
			handlers:               handlers,
		},
		fedClient:       fedClient,
		targetNamespace: namespaces.TargetNamespace,
	}

	// TODO: Update this to use a typed client from single target informer.
	// As of now we have a separate informer for pods, whereas all we need
	// is a typed client.
	// We ignore the pod events in this informer from clusters.
	scheduler.podInformer = NewFederatedInformer(
		fedClient,
		kubeClient,
		crClient,
		namespaces,
		PodResource,
		func(pkgruntime.Object) {},
		handlers,
	)

	return scheduler
}

func (s *ReplicaScheduler) Kind() string {
	return RSPKind
}

func (s *ReplicaScheduler) StartPlugin(kind string, apiResource *metav1.APIResource, stopChan <-chan struct{}) error {
	var adapter adapters.Adapter
	switch kind {
	case FederatedDeployment:
		adapter = adapters.NewFederatedDeploymentAdapter(s.fedClient)
	case FederatedReplicaSet:
		adapter = adapters.NewFederatedReplicaSetAdapter(s.fedClient)
	default:
		return fmt.Errorf("Kind %s is not supported to register as plugin", kind)
	}

	if _, ok := s.plugins[kind]; ok {
		return nil
	}

	glog.Infof("Kind %s is registered to the scheduler plugin", kind)
	s.plugins[kind] = NewPlugin(
		adapter,
		apiResource,
		s.fedClient,
		s.pluginArgs.kubeClient,
		s.pluginArgs.crClient,
		s.pluginArgs.FederationNamespaces,
		s.pluginArgs.federationEventHandler,
		s.pluginArgs.clusterEventHandler,
		s.pluginArgs.handlers,
	)
	s.plugins[kind].Start(stopChan)

	return nil
}

func (s *ReplicaScheduler) ObjectType() pkgruntime.Object {
	return &fedschedulingv1a1.ReplicaSchedulingPreference{}
}

func (s *ReplicaScheduler) FedList(namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return s.fedClient.SchedulingV1alpha1().ReplicaSchedulingPreferences(s.targetNamespace).List(options)
}

func (s *ReplicaScheduler) FedWatch(namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return s.fedClient.SchedulingV1alpha1().ReplicaSchedulingPreferences(s.targetNamespace).Watch(options)
}

func (s *ReplicaScheduler) Start(stopChan <-chan struct{}) {
	s.podInformer.Start()
}

func (s *ReplicaScheduler) HasSynced() bool {
	for _, plugin := range s.plugins {
		if !plugin.HasSynced() {
			return false
		}
	}

	if !s.podInformer.ClustersSynced() {
		glog.V(2).Infof("Cluster list not synced")
		return false
	}
	clusters, err := s.podInformer.GetReadyClusters()
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to get ready clusters: %v", err))
		return false
	}
	return s.podInformer.GetTargetStore().ClustersSynced(clusters)
}

func (s *ReplicaScheduler) Stop() {
	for _, plugin := range s.plugins {
		plugin.Stop()
	}

	s.podInformer.Stop()
}

func (s *ReplicaScheduler) Reconcile(obj pkgruntime.Object, qualifiedName QualifiedName) ReconciliationStatus {
	rsp, ok := obj.(*fedschedulingv1a1.ReplicaSchedulingPreference)
	if !ok {
		runtime.HandleError(fmt.Errorf("Incorrect runtime object for RSP: %v", rsp))
		return StatusError
	}

	clusterNames, err := s.clusterNames()
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to get cluster list: %v", err))
		return StatusError
	}
	if len(clusterNames) == 0 {
		// no joined clusters, nothing to do
		return StatusAllOK
	}

	kind := rsp.Spec.TargetKind
	if kind != FederatedDeployment && kind != FederatedReplicaSet {
		runtime.HandleError(fmt.Errorf("RSP target kind: %s is incorrect: %v", kind, err))
		return StatusNeedsRecheck
	}

	plugin, ok := s.plugins[kind]
	if !ok {
		return StatusAllOK
	}

	if !plugin.TemplateExists(qualifiedName.String()) {
		// target FederatedTemplate does not exist, nothing to do
		return StatusAllOK
	}

	key := qualifiedName.String()
	result, err := s.GetSchedulingResult(rsp, qualifiedName, clusterNames)
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to compute the schedule information while reconciling RSP named %q: %v", key, err))
		return StatusError
	}

	err = s.ReconcileFederationTargets(s.fedClient, qualifiedName, kind, result)
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to reconcile Federation Targets for RSP named %q: %v", key, err))
		return StatusError
	}

	return StatusAllOK
}

// The list of clusters could come from any target informer
func (s *ReplicaScheduler) clusterNames() ([]string, error) {
	clusters, err := s.podInformer.GetReadyClusters()
	if err != nil {
		return nil, err
	}
	clusterNames := []string{}
	for _, cluster := range clusters {
		clusterNames = append(clusterNames, cluster.Name)
	}

	return clusterNames, nil
}

func (s *ReplicaScheduler) ReconcileFederationTargets(fedClient fedclientset.Interface, qualifiedName QualifiedName, kind string, result map[string]int64) error {
	newClusterNames := []string{}
	for name := range result {
		newClusterNames = append(newClusterNames, name)
	}

	err := s.plugins[kind].adapter.ReconcilePlacement(fedClient, qualifiedName, newClusterNames)
	if err != nil {
		return err
	}

	err = s.plugins[kind].adapter.ReconcileOverride(fedClient, qualifiedName, result)
	if err != nil {
		return err
	}

	return nil
}

func (s *ReplicaScheduler) GetSchedulingResult(rsp *fedschedulingv1a1.ReplicaSchedulingPreference, qualifiedName QualifiedName, clusterNames []string) (map[string]int64, error) {
	key := qualifiedName.String()

	objectGetter := func(clusterName, key string) (interface{}, bool, error) {
		return s.plugins[rsp.Spec.TargetKind].targetInformer.GetTargetStore().GetByKey(clusterName, key)
	}
	podsGetter := func(clusterName string, unstructuredObj *unstructured.Unstructured) (pkgruntime.Object, error) {
		client, err := s.podInformer.GetClientForCluster(clusterName)
		if err != nil {
			return nil, err
		}
		selectorLabels, ok, err := unstructured.NestedStringMap(unstructuredObj.Object, "spec", "selector", "matchLabels")
		if !ok {
			return nil, fmt.Errorf("missing selector on object")
		}
		if err != nil {
			return nil, fmt.Errorf("error retrieving selector from object: %v", err)
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
	if len(rsp.Spec.Clusters) == 0 {
		rsp.Spec.Clusters = map[string]fedschedulingv1a1.ClusterPreferences{
			"*": {Weight: 1},
		}
	}

	plnr := planner.NewPlanner(rsp)
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

// clustersReplicaState returns information about the scheduling state of the pods running in the federated clusters.
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
		replicas, ok, err := unstructured.NestedInt64(unstructuredObj.Object, "spec", "replicas")
		if err != nil {
			return nil, nil, fmt.Errorf("Error retrieving 'replicas' field: %v", err)
		}
		if !ok {
			replicas = int64(0)
		}
		readyReplicas, ok, err := unstructured.NestedInt64(unstructuredObj.Object, "status", "readyreplicas")
		if err != nil {
			return nil, nil, fmt.Errorf("Error retrieving 'readyreplicas' field: %v", err)
		}
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
