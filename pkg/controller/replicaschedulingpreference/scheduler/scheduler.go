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

package scheduler

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	fedschedulingv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/federatedscheduling/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset_generated/clientset"
	. "github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util/planner"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util/podanalyzer"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	kubeclientset "k8s.io/client-go/kubernetes"
	crclientset "k8s.io/cluster-registry/pkg/client/clientset_generated/clientset"

	"github.com/golang/glog"
)

const (
	FederatedDeployment = "FederatedDeployment"
	Deployment          = "Deployment"
	FederatedReplicaSet = "FederatedReplicaSet"
	ReplicaSet          = "ReplicaSet"
	Pod                 = "Pod"
)

var resources = map[string]metav1.APIResource{
	FederatedDeployment: {
		Name:       strings.ToLower(Deployment) + "s",
		Group:      appsv1.SchemeGroupVersion.Group,
		Version:    appsv1.SchemeGroupVersion.Version,
		Kind:       Deployment,
		Namespaced: true,
	},
	FederatedReplicaSet: {
		Name:       strings.ToLower(ReplicaSet) + "s",
		Group:      appsv1.SchemeGroupVersion.Group,
		Version:    appsv1.SchemeGroupVersion.Version,
		Kind:       ReplicaSet,
		Namespaced: true,
	},
}

type SchedulerAdapter interface {
	TemplateObject() pkgruntime.Object
	TemplateList(namespace string, options metav1.ListOptions) (pkgruntime.Object, error)
	TemplateWatch(namespace string, options metav1.ListOptions) (watch.Interface, error)

	OverrideObject() pkgruntime.Object
	OverrideList(namespace string, options metav1.ListOptions) (pkgruntime.Object, error)
	OverrideWatch(namespace string, options metav1.ListOptions) (watch.Interface, error)

	PlacementObject() pkgruntime.Object
	PlacementList(namespace string, options metav1.ListOptions) (pkgruntime.Object, error)
	PlacementWatch(namespace string, options metav1.ListOptions) (watch.Interface, error)

	ReconcilePlacement(fedClient fedclientset.Interface, qualifiedName QualifiedName, newClusterNames []string) error
	ReconcileOverride(fedClient fedclientset.Interface, qualifiedName QualifiedName, result map[string]int64) error
}

type Scheduler struct {
	plugins     map[string]*Plugin
	fedClient   fedclientset.Interface
	podInformer FederatedInformer
}

func NewReplicaScheduler(fedClient fedclientset.Interface, kubeClient kubeclientset.Interface, crClient crclientset.Interface, federationEventHandler, clusterEventHandler func(pkgruntime.Object), handlers *ClusterLifecycleHandlerFuncs) *Scheduler {
	scheduler := &Scheduler{}
	scheduler.plugins = make(map[string]*Plugin)
	scheduler.fedClient = fedClient

	for name, apiResource := range resources {
		var adapter SchedulerAdapter
		switch name {
		case FederatedDeployment:
			adapter = NewFederatedDeploymentAdapter(fedClient)
		case FederatedReplicaSet:
			adapter = NewFederatedReplicaSetAdapter(fedClient)
		}
		scheduler.plugins[name] = NewPlugin(
			adapter,
			&apiResource,
			fedClient,
			kubeClient,
			crClient,
			federationEventHandler,
			clusterEventHandler,
			handlers,
		)
	}

	// TODO: Update this to use a typed client from single target informer.
	// As of now we have a separate informer for pods, whereas all we need
	// is a typed client.
	// We ignore the pod events in this informer from clusters.
	scheduler.podInformer = NewFederatedInformer(
		fedClient,
		kubeClient,
		crClient,
		&metav1.APIResource{
			Name:       strings.ToLower(Pod) + "s",
			Group:      corev1.SchemeGroupVersion.Group,
			Version:    corev1.SchemeGroupVersion.Version,
			Kind:       Pod,
			Namespaced: true,
		},
		func(pkgruntime.Object) {},
		handlers,
	)

	return scheduler
}

func (s *Scheduler) Start(stopChan <-chan struct{}) {
	for _, plugin := range s.plugins {
		plugin.Start(stopChan)
	}

	s.podInformer.Start()
}

func (s *Scheduler) HasSynced() bool {
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

func (s *Scheduler) Stop() {
	for _, plugin := range s.plugins {
		plugin.Stop()
	}

	s.podInformer.Stop()
}

func (s *Scheduler) Reconcile(rsp *fedschedulingv1a1.ReplicaSchedulingPreference, qualifiedName QualifiedName) ReconciliationStatus {
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

	if !s.plugins[kind].TemplateExists(qualifiedName.String()) {
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
func (s *Scheduler) clusterNames() ([]string, error) {
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

func (s *Scheduler) ReconcileFederationTargets(fedClient fedclientset.Interface, qualifiedName QualifiedName, kind string, result map[string]int64) error {
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

func (s *Scheduler) GetSchedulingResult(rsp *fedschedulingv1a1.ReplicaSchedulingPreference, qualifiedName QualifiedName, clusterNames []string) (map[string]int64, error) {
	key := qualifiedName.String()

	objectGetter := func(clusterName, key string) (interface{}, bool, error) {
		return s.plugins[rsp.Spec.TargetKind].targetInformer.GetTargetStore().GetByKey(clusterName, key)
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
