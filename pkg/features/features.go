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
	// PushReconciler is a propogation model where in objects are pushed to member clusters from federation.
	PushReconciler utilfeature.Feature = "PushReconciler"

	// owner: @irfanurrehman
	// alpha: v0.1
	//
	// Scheduler controllers which dynamically schedules workloads based on user preferences.
	SchedulerPreferences utilfeature.Feature = "SchedulerPreferences"

	// owner: @kubernetes-sigs/federation-v2-maintainers
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
	utilfeature.DefaultFeatureGate.Add(defaultFederationFeatureGates)
}

// defaultFederationFeatureGates consists of all known Federation-specific feature keys.
// To add a new feature, define a key for it above and add it here. The features will be
// available throughout Federation binaries.
var defaultFederationFeatureGates = map[utilfeature.Feature]utilfeature.FeatureSpec{
	SchedulerPreferences:         {Default: true, PreRelease: utilfeature.Alpha},
	PushReconciler:               {Default: true, PreRelease: utilfeature.Alpha},
	CrossClusterServiceDiscovery: {Default: true, PreRelease: utilfeature.Alpha},
	FederatedIngress:             {Default: true, PreRelease: utilfeature.Alpha},
}
