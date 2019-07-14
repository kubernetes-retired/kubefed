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

package features

import (
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/klog"
)

const (
	// Every feature gate should add method here following this template:
	//
	// // owner: @username
	// // alpha: v1.X
	// MyFeature utilfeature.Feature = "MyFeature"

	// owner: @marun
	// alpha: v0.1
	//
	// PushReconciler ensures that managed resources in member clusters represent the state declared in federated resources.
	PushReconciler utilfeature.Feature = "PushReconciler"

	// owner: @irfanurrehman
	// alpha: v0.1
	//
	// Scheduler controllers which dynamically schedules workloads based on user preferences.
	SchedulerPreferences utilfeature.Feature = "SchedulerPreferences"

	// owner: @kubernetes-sigs/kubefed-maintainers
	// alpha: v0.1
	//
	// DNS based cross cluster service discovery.
	// https://github.com/kubernetes/community/blob/master/contributors/design-proposals/multicluster/federated-services.md
	CrossClusterServiceDiscovery utilfeature.Feature = "CrossClusterServiceDiscovery"

	// owner: @shashidharatd
	// alpha: v0.1
	//
	// DNS based federated ingress feature.
	FederatedIngress utilfeature.Feature = "FederatedIngress"
)

func init() {
	if err := utilfeature.DefaultFeatureGate.Add(DefaultKubeFedFeatureGates); err != nil {
		klog.Fatalf("Unexpected error: %v", err)
	}
}

// DefaultKubeFedFeatureGates consists of all known KubeFed-specific
// feature keys.  To add a new feature, define a key for it above and
// add it here.
var DefaultKubeFedFeatureGates = map[utilfeature.Feature]utilfeature.FeatureSpec{
	SchedulerPreferences:         {Default: true, PreRelease: utilfeature.Alpha},
	PushReconciler:               {Default: true, PreRelease: utilfeature.Beta},
	CrossClusterServiceDiscovery: {Default: true, PreRelease: utilfeature.Alpha},
	FederatedIngress:             {Default: true, PreRelease: utilfeature.Alpha},
}
