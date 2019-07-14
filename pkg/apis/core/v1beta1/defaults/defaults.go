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
	"sort"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	"sigs.k8s.io/kubefed/pkg/features"
)

const (
	DefaultClusterAvailableDelay   = 20 * time.Second
	DefaultClusterUnavailableDelay = 60 * time.Second

	DefaultLeaderElectionLeaseDuration = 15 * time.Second
	DefaultLeaderElectionRenewDeadline = 10 * time.Second
	DefaultLeaderElectionRetryPeriod   = 5 * time.Second
	DefaultLeaderElectionResourceLock  = v1beta1.ConfigMapsResourceLock

	DefaultClusterHealthCheckPeriod           = 10 * time.Second
	DefaultClusterHealthCheckFailureThreshold = 3
	DefaultClusterHealthCheckSuccessThreshold = 1
	DefaultClusterHealthCheckTimeout          = 3 * time.Second
)

func SetDefaultKubeFedConfig(fedConfig *v1beta1.KubeFedConfig) {
	spec := &fedConfig.Spec

	if spec.ControllerDuration == nil {
		spec.ControllerDuration = &v1beta1.DurationConfig{}
	}

	duration := spec.ControllerDuration
	setDuration(&duration.AvailableDelay, DefaultClusterAvailableDelay)
	setDuration(&duration.UnavailableDelay, DefaultClusterUnavailableDelay)

	if spec.LeaderElect == nil {
		spec.LeaderElect = &v1beta1.LeaderElectConfig{}
	}

	election := spec.LeaderElect

	if election.ResourceLock == nil {
		election.ResourceLock = new(v1beta1.ResourceLockType)
		*election.ResourceLock = DefaultLeaderElectionResourceLock
	}

	setDuration(&election.RetryPeriod, DefaultLeaderElectionRetryPeriod)
	setDuration(&election.RenewDeadline, DefaultLeaderElectionRenewDeadline)
	setDuration(&election.LeaseDuration, DefaultLeaderElectionLeaseDuration)

	if spec.FeatureGates == nil {
		spec.FeatureGates = make([]v1beta1.FeatureGatesConfig, 0)
	}

	spec.FeatureGates = setDefaultKubeFedFeatureGates(spec.FeatureGates)

	if spec.ClusterHealthCheck == nil {
		spec.ClusterHealthCheck = &v1beta1.ClusterHealthCheckConfig{}
	}

	healthCheck := spec.ClusterHealthCheck
	setDuration(&healthCheck.Period, DefaultClusterHealthCheckPeriod)
	setDuration(&healthCheck.Timeout, DefaultClusterHealthCheckTimeout)
	setInt64(&healthCheck.FailureThreshold, DefaultClusterHealthCheckFailureThreshold)
	setInt64(&healthCheck.SuccessThreshold, DefaultClusterHealthCheckSuccessThreshold)

	if spec.SyncController == nil {
		spec.SyncController = &v1beta1.SyncControllerConfig{}
	}

	if spec.SyncController.AdoptResources == nil {
		spec.SyncController.AdoptResources = new(v1beta1.ResourceAdoption)
		*spec.SyncController.AdoptResources = v1beta1.AdoptResourcesEnabled
	}
}

func setDefaultKubeFedFeatureGates(fgc []v1beta1.FeatureGatesConfig) []v1beta1.FeatureGatesConfig {
	for defaultFeatureName, spec := range features.DefaultKubeFedFeatureGates {
		useDefault := true
		for _, configFeature := range fgc {
			if string(defaultFeatureName) == configFeature.Name {
				useDefault = false
			}
		}

		if !useDefault {
			continue
		}

		configuration := v1beta1.ConfigurationEnabled
		if !spec.Default {
			configuration = v1beta1.ConfigurationDisabled
		}

		fgc = append(fgc, v1beta1.FeatureGatesConfig{
			Name:          string(defaultFeatureName),
			Configuration: configuration,
		})
	}

	// Since we iterated over a map of default feature gates, this code
	// sets the default features in a deterministic order within the
	// FeatureGatesConfig slice.
	sort.Slice(fgc[:], func(i, j int) bool {
		return fgc[i].Name < fgc[j].Name
	})

	return fgc
}

func setDuration(target **metav1.Duration, defaultValue time.Duration) {
	if *target == nil {
		*target = &metav1.Duration{}
		(*target).Duration = defaultValue
	}
}

func setInt64(target **int64, defaultValue int64) {
	if *target == nil {
		*target = new(int64)
		**target = defaultValue
	}
}
