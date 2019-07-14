/*
Copyright 2019 The Kubernetes Authors.

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

package defaults

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	apiextv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	"sigs.k8s.io/kubefed/pkg/controller/util"
)

func TestSetDefaultKubeFedConfig(t *testing.T) {
	type KubeFedConfigComparison struct {
		original *v1beta1.KubeFedConfig
		modified *v1beta1.KubeFedConfig
	}

	successCases := map[string]KubeFedConfigComparison{}

	// ControllerDuration
	availableDelayKFC := defaultKubeFedConfig()
	availableDelayKFC.Spec.ControllerDuration.AvailableDelay.Duration = DefaultClusterAvailableDelay + 31*time.Second
	modifiedAvailableDelayKFC := availableDelayKFC.DeepCopyObject().(*v1beta1.KubeFedConfig)
	SetDefaultKubeFedConfig(modifiedAvailableDelayKFC)
	successCases["spec.controllerDuration.availableDelay is preserved"] = KubeFedConfigComparison{availableDelayKFC, modifiedAvailableDelayKFC}

	unavailableDelayKFC := defaultKubeFedConfig()
	unavailableDelayKFC.Spec.ControllerDuration.UnavailableDelay.Duration = DefaultClusterUnavailableDelay + 31*time.Second
	modifiedUnavailableDelayKFC := unavailableDelayKFC.DeepCopyObject().(*v1beta1.KubeFedConfig)
	SetDefaultKubeFedConfig(modifiedUnavailableDelayKFC)
	successCases["spec.controllerDuration.unavailableDelay is preserved"] = KubeFedConfigComparison{unavailableDelayKFC, modifiedUnavailableDelayKFC}

	// LeaderElect
	leaseDurationKFC := defaultKubeFedConfig()
	leaseDurationKFC.Spec.LeaderElect.LeaseDuration.Duration = DefaultLeaderElectionLeaseDuration + 11*time.Second
	modifiedLeaseDurationKFC := leaseDurationKFC.DeepCopyObject().(*v1beta1.KubeFedConfig)
	SetDefaultKubeFedConfig(modifiedLeaseDurationKFC)
	successCases["spec.leaderElect.leaseDuration is preserved"] = KubeFedConfigComparison{leaseDurationKFC, modifiedLeaseDurationKFC}

	renewDeadlineKFC := defaultKubeFedConfig()
	renewDeadlineKFC.Spec.LeaderElect.RenewDeadline.Duration = DefaultLeaderElectionRenewDeadline + 11*time.Second
	modifiedRenewDeadlineKFC := renewDeadlineKFC.DeepCopyObject().(*v1beta1.KubeFedConfig)
	SetDefaultKubeFedConfig(modifiedRenewDeadlineKFC)
	successCases["spec.leaderElect.renewDeadline is preserved"] = KubeFedConfigComparison{renewDeadlineKFC, modifiedRenewDeadlineKFC}

	retryPeriodKFC := defaultKubeFedConfig()
	retryPeriodKFC.Spec.LeaderElect.RetryPeriod.Duration = DefaultLeaderElectionRetryPeriod + 11*time.Second
	modifiedRetryPeriodKFC := retryPeriodKFC.DeepCopyObject().(*v1beta1.KubeFedConfig)
	SetDefaultKubeFedConfig(modifiedRetryPeriodKFC)
	successCases["spec.leaderElect.retryPeriod is preserved"] = KubeFedConfigComparison{retryPeriodKFC, modifiedRetryPeriodKFC}

	resourceLockKFC := defaultKubeFedConfig()
	*resourceLockKFC.Spec.LeaderElect.ResourceLock = v1beta1.EndpointsResourceLock
	modifiedResourceLockKFC := resourceLockKFC.DeepCopyObject().(*v1beta1.KubeFedConfig)
	SetDefaultKubeFedConfig(modifiedResourceLockKFC)
	successCases["spec.leaderElect.resourceLock is preserved"] = KubeFedConfigComparison{resourceLockKFC, modifiedResourceLockKFC}

	// FeatureGates
	defaultFeatureGates := defaultKubeFedConfig()
	for i, featureGate := range defaultFeatureGates.Spec.FeatureGates {
		featureGatesKFC := defaultFeatureGates.DeepCopyObject().(*v1beta1.KubeFedConfig)
		// Toggle Configuration
		if featureGate.Configuration == v1beta1.ConfigurationEnabled {
			featureGatesKFC.Spec.FeatureGates[i].Configuration = v1beta1.ConfigurationDisabled
		} else {
			featureGatesKFC.Spec.FeatureGates[i].Configuration = v1beta1.ConfigurationEnabled
		}

		modifiedFeatureGatesKFC := featureGatesKFC.DeepCopyObject().(*v1beta1.KubeFedConfig)
		SetDefaultKubeFedConfig(modifiedFeatureGatesKFC)
		caseName := fmt.Sprintf("spec.featureGates.%s is preserved", featureGate.Name)
		successCases[caseName] = KubeFedConfigComparison{featureGatesKFC, modifiedFeatureGatesKFC}
	}

	// ClusterHealthCheck
	periodKFC := defaultKubeFedConfig()
	periodKFC.Spec.ClusterHealthCheck.Period.Duration = DefaultClusterHealthCheckPeriod + 11*time.Second
	modifiedPeriodKFC := periodKFC.DeepCopyObject().(*v1beta1.KubeFedConfig)
	SetDefaultKubeFedConfig(modifiedPeriodKFC)
	successCases["spec.clusterHealthCheck.period is preserved"] = KubeFedConfigComparison{periodKFC, modifiedPeriodKFC}

	failureThresholdKFC := defaultKubeFedConfig()
	failureThreshold := int64(DefaultClusterHealthCheckFailureThreshold + 5)
	failureThresholdKFC.Spec.ClusterHealthCheck.FailureThreshold = &failureThreshold
	modifiedFailureThresholdKFC := failureThresholdKFC.DeepCopyObject().(*v1beta1.KubeFedConfig)
	SetDefaultKubeFedConfig(modifiedFailureThresholdKFC)
	successCases["spec.clusterHealthCheck.failureThreshold is preserved"] = KubeFedConfigComparison{failureThresholdKFC, modifiedFailureThresholdKFC}

	successThresholdKFC := defaultKubeFedConfig()
	successThreshold := int64(DefaultClusterHealthCheckSuccessThreshold + 3)
	successThresholdKFC.Spec.ClusterHealthCheck.SuccessThreshold = &successThreshold
	modifiedSuccessThresholdKFC := successThresholdKFC.DeepCopyObject().(*v1beta1.KubeFedConfig)
	SetDefaultKubeFedConfig(modifiedSuccessThresholdKFC)
	successCases["spec.clusterHealthCheck.successThreshold is preserved"] = KubeFedConfigComparison{successThresholdKFC, modifiedSuccessThresholdKFC}

	timeoutKFC := defaultKubeFedConfig()
	timeoutKFC.Spec.ClusterHealthCheck.Timeout.Duration = DefaultClusterHealthCheckTimeout + 13*time.Second
	modifiedTimeoutKFC := timeoutKFC.DeepCopyObject().(*v1beta1.KubeFedConfig)
	SetDefaultKubeFedConfig(modifiedTimeoutKFC)
	successCases["spec.clusterHealthCheck.timeout is preserved"] = KubeFedConfigComparison{timeoutKFC, modifiedTimeoutKFC}

	// SyncController
	adoptResourcesKFC := defaultKubeFedConfig()
	*adoptResourcesKFC.Spec.SyncController.AdoptResources = v1beta1.AdoptResourcesDisabled
	modifiedAdoptResourcesKFC := adoptResourcesKFC.DeepCopyObject().(*v1beta1.KubeFedConfig)
	SetDefaultKubeFedConfig(modifiedAdoptResourcesKFC)
	successCases["spec.leaderElect.adoptResources is preserved"] = KubeFedConfigComparison{adoptResourcesKFC, modifiedAdoptResourcesKFC}

	for k, v := range successCases {
		if !reflect.DeepEqual(v.original, v.modified) {
			t.Errorf("[%s] expected success: original=%+v, modified=%+v", k, *v.original, *v.modified)
		}
	}
}

func defaultKubeFedConfig() *v1beta1.KubeFedConfig {
	kfc := &v1beta1.KubeFedConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: util.DefaultKubeFedSystemNamespace,
			Name:      util.KubeFedConfigName,
		},
		Spec: v1beta1.KubeFedConfigSpec{
			Scope: apiextv1b1.ClusterScoped,
		},
	}

	SetDefaultKubeFedConfig(kfc)
	return kfc
}
