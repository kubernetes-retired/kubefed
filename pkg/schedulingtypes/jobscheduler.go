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
	"fmt"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	fedschedulingv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/scheduling/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	. "github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util/planner"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

const (
	JobSchedulingKind = "JobSchedulingPreference"
	parallelismField  = "parallelism"
	completionsField  = "completions"
)

func init() {
	schedulingType := SchedulingType{
		Kind:             JobSchedulingKind,
		SchedulerFactory: NewJobScheduler,
	}
	RegisterSchedulingType("jobs.batch", schedulingType)
}

type JobScheduler struct {
	plugin *Plugin

	fedClient fedclientset.Interface

	controllerConfig *ControllerConfig

	eventHandlers SchedulerEventHandlers
}

func NewJobScheduler(controllerConfig *ControllerConfig, eventHandlers SchedulerEventHandlers) (Scheduler, error) {
	fedClient, _, _ := controllerConfig.AllClients("job-scheduler")
	scheduler := &JobScheduler{
		controllerConfig: controllerConfig,
		eventHandlers:    eventHandlers,
		fedClient:        fedClient,
	}

	return scheduler, nil
}

func (j *JobScheduler) Kind() string {
	return RSPKind
}

func (j *JobScheduler) StartPlugin(typeConfig typeconfig.Interface, stopChan <-chan struct{}) error {
	kind := typeConfig.GetTemplate().Kind
	plugin, err := NewPlugin(j.controllerConfig, j.eventHandlers, typeConfig)
	if err != nil {
		return fmt.Errorf("Failed to initialize replica scheduling plugin for %q: %v", kind, err)
	}
	plugin.Start(stopChan)
	j.plugin = plugin

	return nil
}

func (j *JobScheduler) ObjectType() pkgruntime.Object {
	return &fedschedulingv1a1.JobSchedulingPreference{}
}

func (j *JobScheduler) FedList(namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return j.fedClient.SchedulingV1alpha1().JobSchedulingPreferences(j.controllerConfig.TargetNamespace).List(options)
}

func (j *JobScheduler) FedWatch(namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return j.fedClient.SchedulingV1alpha1().JobSchedulingPreferences(j.controllerConfig.TargetNamespace).Watch(options)
}

func (j *JobScheduler) Start(stopChan <-chan struct{}) {
	j.plugin.Start(stopChan)
}

func (j *JobScheduler) HasSynced() bool {
	return j.plugin.HasSynced()
}

func (j *JobScheduler) Stop() {
	j.plugin.Stop()
}

func (j *JobScheduler) Reconcile(obj pkgruntime.Object, qualifiedName QualifiedName) ReconciliationStatus {
	jsp, ok := obj.(*fedschedulingv1a1.JobSchedulingPreference)
	if !ok {
		runtime.HandleError(fmt.Errorf("Incorrect runtime object for JSP: %v", jsp))
		return StatusError
	}

	clusterNames, err := j.plugin.ReadyClusterNames()
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to get cluster list: %v", err))
		return StatusError
	}
	if len(clusterNames) == 0 {
		// no joined clusters, nothing to do
		return StatusAllOK
	}

	result := j.schedule(jsp, qualifiedName, clusterNames)
	err = j.ReconcileFederationTargets(j.fedClient, qualifiedName, result)
	if err != nil {
		runtime.HandleError(fmt.Errorf("Failed to reconcile Federation Targets for JSP named %s: %v", qualifiedName, err))
		return StatusError
	}

	return StatusAllOK
}

func (j *JobScheduler) ReconcileFederationTargets(fedClient fedclientset.Interface, qualifiedName QualifiedName, result *JobScheduleResult) error {
	err := j.plugin.ReconcilePlacement(qualifiedName, result)
	if err != nil {
		return err
	}

	err = j.plugin.ReconcileOverride(qualifiedName, result)
	if err != nil {
		return err
	}

	return nil
}

type ClusterJobValues struct {
	Parallelism int32
	Completions int32
}

type JobScheduleResult struct {
	result map[string]ClusterJobValues
}

func (r *JobScheduleResult) clusterNames() []string {
	clusterNames := []string{}
	for name := range r.result {
		clusterNames = append(clusterNames, name)
	}
	return clusterNames
}

func (r *JobScheduleResult) SetPlacementSpec(obj *unstructured.Unstructured) {
	obj.Object[util.SpecField] = map[string]interface{}{
		util.ClusterNamesField: r.clusterNames(),
	}
}

// TODO (irfanurrehman): in this pass we ignore updates for jobs
func (r *JobScheduleResult) PlacementUpdateNeeded(names []string) bool {
	return false
}

func (r *JobScheduleResult) SetOverrideSpec(obj *unstructured.Unstructured) {
	overrides := []interface{}{}
	for clusterName, values := range r.result {
		overridesMap := map[string]interface{}{
			util.ClusterNameField: clusterName,
			parallelismField:      values.Parallelism,
			completionsField:      values.Completions,
		}
		overrides = append(overrides, overridesMap)
	}
	obj.Object[util.SpecField] = map[string]interface{}{
		util.OverridesField: overrides,
	}
}

// TODO (irfanurrehman): in this pass we ignore updates for jobs
func (r *JobScheduleResult) OverrideUpdateNeeded(typeConfig typeconfig.Interface, obj *unstructured.Unstructured) bool {
	return false
}

func (j *JobScheduler) schedule(jsp *fedschedulingv1a1.JobSchedulingPreference, qualifiedName QualifiedName, clusterNames []string) *JobScheduleResult {
	key := qualifiedName.String()
	if len(jsp.Spec.ClusterWeights) == 0 {
		jsp.Spec.ClusterWeights = map[string]int32{
			"*": 1,
		}
	}

	plnr := planner.NewJobPlanner(jsp)
	parallelismResult := plnr.Plan(clusterNames, jsp.Spec.TotalParallelism, key)

	clusterNames = nil
	for clusterName := range parallelismResult {
		clusterNames = append(clusterNames, clusterName)
	}
	completionsResult := plnr.Plan(clusterNames, jsp.Spec.TotalCompletions, key)

	result := make(map[string]ClusterJobValues)
	for _, clusterName := range clusterNames {
		result[clusterName] = ClusterJobValues{
			Parallelism: parallelismResult[clusterName],
			Completions: completionsResult[clusterName],
		}
	}

	return &JobScheduleResult{
		result: result,
	}
}
