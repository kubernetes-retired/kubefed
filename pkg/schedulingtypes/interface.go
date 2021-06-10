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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/kubefed/pkg/apis/core/typeconfig"
	"sigs.k8s.io/kubefed/pkg/controller/util"
)

type Scheduler interface {
	SchedulingKind() string
	ObjectType() runtimeclient.Object

	Start()
	HasSynced() bool
	Stop()
	Reconcile(obj runtimeclient.Object, qualifiedName util.QualifiedName) util.ReconciliationStatus

	StartPlugin(typeConfig typeconfig.Interface, nsAPIResource *metav1.APIResource) error
	StopPlugin(kind string)
}

type SchedulerEventHandlers struct {
	KubeFedEventHandler      func(runtimeclient.Object)
	ClusterEventHandler      func(runtimeclient.Object)
	ClusterLifecycleHandlers *util.ClusterLifecycleHandlerFuncs
}

type SchedulerFactory func(controllerConfig *util.ControllerConfig, eventHandlers SchedulerEventHandlers) (Scheduler, error)
