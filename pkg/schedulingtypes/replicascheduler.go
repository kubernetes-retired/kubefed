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

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
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

	"github.com/golang/glog"
)

const (
	RSPKind = "ReplicaSchedulingPreference"
)

func init() {
	schedulingType := SchedulingType{
		Kind:             RSPKind,
		SchedulerFactory: NewReplicaScheduler,
	}
	RegisterSchedulingType("deployments.apps", schedulingType)
	RegisterSchedulingType("replicasets.apps", schedulingType)
}

type ReplicaScheduler struct {
	controllerConfig *ControllerConfig

	eventHandlers SchedulerEventHandlers

	plugins map[string]*Plugin

	fedClient   fedclientset.Interface
	podInformer FederatedInformer
}

func NewReplicaScheduler(controllerConfig *ControllerConfig, eventHandlers SchedulerEventHandlers) (Scheduler, error) {
	fedClient, kubeClient, crClient := controllerConfig.AllClients("replica-scheduler")
	scheduler := &ReplicaScheduler{
		plugins:          make(map[string]*Plugin),
		controllerConfig: controllerConfig,
		eventHandlers:    eventHandlers,
		fedClient:        fedClient,
	}

	// TODO: Update this to use a typed client from single target informer.
	// As of now we have a separate informer for pods, whereas all we need
	// is a typed client.
	// We ignore the pod events in this informer from clusters.
	scheduler.podInformer = NewFederatedInformer(
		fedClient,
		kubeClient,
		crClient,
		controllerConfig.FederationNamespaces,
		PodResource,
		func(pkgruntime.Object) {},
		eventHandlers.ClusterLifecycleHandlers,
	)

	return scheduler, nil
}

func (s *ReplicaScheduler) Kind() string {
	return RSPKind
}

func (s *ReplicaScheduler) StartPlugin(typeConfig typeconfig.Interface, stopChan <-chan struct{}) error {
	kind := typeConfig.GetTemplate().Kind
	// TODO(marun) Return an error if the kind is not supported

	plugin, err := NewPlugin(s.controllerConfig, s.eventHandlers, typeConfig)
	if err != nil {
		return fmt.Errorf("Failed to initialize replica scheduling plugin for %q: %v", kind, err)
	}
	plugin.Start(stopChan)
	s.plugins[kind] = plugin

	return nil
}

func (s *ReplicaScheduler) ObjectType() pkgruntime.Object {
	return &fedschedulingv1a1.ReplicaSchedulingPreference{}
}

func (s *ReplicaScheduler) FedList(namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return s.fedClient.SchedulingV1alpha1().ReplicaSchedulingPreferences(s.controllerConfig.TargetNamespace).List(options)
}

func (s *ReplicaScheduler) FedWatch(namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return s.fedClient.SchedulingV1alpha1().ReplicaSchedulingPreferences(s.controllerConfig.TargetNamespace).Watch(options)
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
	if kind != "FederatedDeployment" && kind != "FederatedReplicaSet" {
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

	err = s.ReconcileFederationTargets(qualifiedName, kind, result)
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

func (s *ReplicaScheduler) ReconcileFederationTargets(qualifiedName QualifiedName, kind string, result map[string]int64) error {
	newClusterNames := []string{}
	for name := range result {
		newClusterNames = append(newClusterNames, name)
	}

	err := s.plugins[kind].ReconcilePlacement(qualifiedName, newClusterNames)
	if err != nil {
		return err
	}

	err = s.plugins[kind].ReconcileOverride(qualifiedName, result)
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
