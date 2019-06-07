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

package validation

import (
	"fmt"
	"strings"
	"testing"

	apiextv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"sigs.k8s.io/kubefed/cmd/controller-manager/app"
	"sigs.k8s.io/kubefed/pkg/apis/core/typeconfig"
	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	"sigs.k8s.io/kubefed/pkg/controller/util"
	"sigs.k8s.io/kubefed/pkg/features"
	"sigs.k8s.io/kubefed/pkg/kubefedctl/enable"
	"sigs.k8s.io/kubefed/pkg/kubefedctl/options"
)

func TestValidateFederatedTypeConfig(t *testing.T) {
	statusSubResource := []bool{true, false}
	for _, status := range statusSubResource {
		errs := ValidateFederatedTypeConfig(validFederatedTypeConfig(), status)
		if len(errs) != 0 {
			t.Errorf("expected success: %v", errs)
		}
	}
}

func TestValidateFederatedTypeConfigName(t *testing.T) {
	for _, successCase := range successCases() {
		if errs := ValidateFederatedTypeConfigName(successCase); len(errs) != 0 {
			t.Errorf("expected success: %v", errs)
		}
	}

	errorCases := map[string]*v1beta1.FederatedTypeConfig{}

	validDeploymentFedTypeConfig := federatedTypeConfig(apiResourceWithNonEmptyGroup())
	validDeploymentFedTypeConfig.Name = "deployments"
	errorCases[federatedTypeConfigNameErrorMsg] = validDeploymentFedTypeConfig

	validServicesFedTypeConfig := federatedTypeConfig(apiResourceWithEmptyGroup())
	validServicesFedTypeConfig.Name = "service"
	errorCases["name must be 'TARGET_PLURAL_NAME"] = validServicesFedTypeConfig

	for k, v := range errorCases {
		errs := ValidateFederatedTypeConfigName(v)
		if len(errs) == 0 {
			t.Errorf("[%s] expected failure", k)
		} else if !strings.Contains(errs[0].Error(), k) {
			t.Errorf("unexpected error: %q, expected: %q", errs[0].Error(), k)
		}
	}
}

func TestValidateFederatedTypeConfigSpec(t *testing.T) {
	for _, successCase := range successCases() {
		if errs := ValidateFederatedTypeConfigSpec(&successCase.Spec, field.NewPath("spec")); len(errs) != 0 {
			t.Errorf("expected success: %v", errs)
		}
	}

	errorCases := map[string]*v1beta1.FederatedTypeConfig{}

	// Validate required fields
	fedGroupRequired := validFederatedTypeConfig()
	fedGroupRequired.Spec.FederatedType.Group = ""
	errorCases["federatedType.group: Required value"] = fedGroupRequired

	versionRequired := validFederatedTypeConfig()
	versionRequired.Spec.TargetType.Version = ""
	errorCases["targetType.version: Required value"] = versionRequired

	kindRequired := validFederatedTypeConfig()
	kindRequired.Spec.TargetType.Kind = ""
	errorCases["targetType.kind: Required value"] = kindRequired

	pluralName := validFederatedTypeConfig()
	pluralName.Spec.TargetType.PluralName = ""
	errorCases["targetType.pluralName: Required value"] = pluralName

	scope := validFederatedTypeConfig()
	scope.Spec.TargetType.Scope = ""
	errorCases["targetType.scope: Required value"] = scope

	propagation := validFederatedTypeConfig()
	propagation.Spec.Propagation = ""
	errorCases["spec.propagation: Required value"] = propagation

	// Validate field values
	validFedGroup := validFederatedTypeConfig()
	validFedGroup.Spec.FederatedType.Group = "nodomain"
	errorCases[domainWithAtLeastOneDot] = validFedGroup

	validTargetGroup := validFederatedTypeConfig()
	validTargetGroup.Spec.TargetType.Group = "invalid#group"
	errorCases["consist of lower case alphanumeric characters, '-' or '.'"] = validTargetGroup

	validVersion := validFederatedTypeConfig()
	validVersion.Spec.TargetType.Version = "Alpha"
	errorCases["must consist of lower case alphanumeric characters"] = validVersion

	validKind := validFederatedTypeConfig()
	validKind.Spec.TargetType.Kind = "Invalid.Kind"
	errorCases["alphanumeric characters or '-'"] = validKind

	validPluralName := validFederatedTypeConfig()
	validPluralName.Spec.TargetType.PluralName = "2InvalidKind"
	errorCases["start with an alphabetic character"] = validPluralName

	validScope := validFederatedTypeConfig()
	validScope.Spec.TargetType.Scope = "NeitherClusterOrNamespaceScoped"
	errorCases["targetType.scope: Unsupported value"] = validScope

	validPropagation := validFederatedTypeConfig()
	validPropagation.Spec.Propagation = "InvalidPropagationMode"
	errorCases["spec.propagation: Unsupported value"] = validPropagation

	validStatusCollection := validFederatedTypeConfig()
	var invalidStatusCollectionMode v1beta1.StatusCollectionMode = "InvalidStatusCollectionMode"
	validStatusCollection.Spec.StatusCollection = &invalidStatusCollectionMode
	errorCases["spec.statusCollection: Unsupported value"] = validStatusCollection

	for k, v := range errorCases {
		errs := ValidateFederatedTypeConfigSpec(&v.Spec, field.NewPath("spec"))
		if len(errs) == 0 {
			t.Errorf("[%s] expected failure", k)
		} else if !strings.Contains(errs[0].Error(), k) {
			t.Errorf("unexpected error: %q, expected: %q", errs[0].Error(), k)
		}
	}
}

func TestValidateFederatedTypeConfigStatus(t *testing.T) {
	running := v1beta1.ControllerStatusRunning
	notRunning := v1beta1.ControllerStatusNotRunning
	var invalidControllerStatus v1beta1.ControllerStatus = "InvalidControllerStatus"
	testCases := []struct {
		name                  string
		observedGeneration    int64
		propagationController v1beta1.ControllerStatus
		statusController      *v1beta1.ControllerStatus
		expectedErr           bool
		expectedErrMsg        string
	}{
		{
			name:                  "valid status",
			observedGeneration:    1,
			propagationController: running,
			statusController:      &notRunning,
			expectedErr:           false,
		},
		{
			name:               "PropagationController required",
			observedGeneration: 1,
			statusController:   &running,
			expectedErr:        true,
			expectedErrMsg:     "status.propagationController: Required value",
		},
		{
			name:                  "negative ObservedGeneration",
			observedGeneration:    -1,
			propagationController: running,
			statusController:      &notRunning,
			expectedErr:           true,
			expectedErrMsg:        "must be greater than or equal to 0",
		},
		{
			name:                  "invalid PropagationController value",
			observedGeneration:    1,
			propagationController: invalidControllerStatus,
			statusController:      &running,
			expectedErr:           true,
			expectedErrMsg:        "status.propagationController: Unsupported value",
		},
		{
			name:                  "invalid StatusController value",
			observedGeneration:    1,
			propagationController: running,
			statusController:      &invalidControllerStatus,
			expectedErr:           true,
			expectedErrMsg:        "status.statusController: Unsupported value",
		},
	}

	for _, test := range testCases {
		status := &v1beta1.FederatedTypeConfigStatus{
			ObservedGeneration:    test.observedGeneration,
			PropagationController: test.propagationController,
			StatusController:      test.statusController,
		}

		errs := ValidateFederatedTypeConfigStatus(status, field.NewPath("status"))
		hasErr := len(errs) > 0
		if hasErr && hasErr != test.expectedErr {
			t.Errorf("[%s] expected failure", test.expectedErrMsg)
		} else if hasErr && !strings.Contains(errs[0].Error(), test.expectedErrMsg) {
			t.Errorf("unexpected error: %q, expected: %q", errs[0].Error(), test.expectedErrMsg)
		}
	}

}

func successCases() []*v1beta1.FederatedTypeConfig {
	return []*v1beta1.FederatedTypeConfig{
		federatedTypeConfig(apiResourceWithEmptyGroup()),
		federatedTypeConfig(apiResourceWithNonEmptyGroup()),
	}
}

func apiResourceWithEmptyGroup() *metav1.APIResource {
	return &metav1.APIResource{
		Group:      "",
		Version:    "v1",
		Kind:       "Service",
		Name:       "services",
		Namespaced: true,
	}
}

func apiResourceWithNonEmptyGroup() *metav1.APIResource {
	return &metav1.APIResource{
		Group:      "apps",
		Version:    "v1",
		Kind:       "Deployment",
		Name:       "deployments",
		Namespaced: true,
	}
}

func validFederatedTypeConfig() *v1beta1.FederatedTypeConfig {
	return federatedTypeConfig(apiResourceWithNonEmptyGroup())
}

func federatedTypeConfig(apiResource *metav1.APIResource) *v1beta1.FederatedTypeConfig {
	kind := apiResource.Kind
	pluralName := apiResource.Name
	statusCollection := v1beta1.StatusCollectionEnabled
	statusController := v1beta1.ControllerStatusNotRunning
	ftc := &v1beta1.FederatedTypeConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: typeconfig.GroupQualifiedName(*apiResource),
		},
		Spec: v1beta1.FederatedTypeConfigSpec{
			TargetType: v1beta1.APIResource{
				Group:      apiResource.Group,
				Version:    apiResource.Version,
				Kind:       kind,
				PluralName: pluralName,
				Scope:      enable.NamespacedToScope(*apiResource),
			},
			Propagation: v1beta1.PropagationEnabled,
			FederatedType: v1beta1.APIResource{
				Group:      options.DefaultFederatedGroup,
				Version:    options.DefaultFederatedVersion,
				Kind:       fmt.Sprintf("Federated%s", kind),
				PluralName: fmt.Sprintf("federated%s", pluralName),
				Scope:      enable.FederatedNamespacedToScope(*apiResource),
			},
			StatusType: &v1beta1.APIResource{
				Group:      options.DefaultFederatedGroup,
				Version:    options.DefaultFederatedVersion,
				Kind:       fmt.Sprintf("Federated%sStatus", kind),
				PluralName: fmt.Sprintf("federated%sstatus", pluralName),
				Scope:      enable.FederatedNamespacedToScope(*apiResource),
			},
			StatusCollection: &statusCollection,
		},
		Status: v1beta1.FederatedTypeConfigStatus{
			ObservedGeneration:    1,
			PropagationController: v1beta1.ControllerStatusRunning,
			StatusController:      &statusController,
		},
	}
	return ftc
}

func TestValidateKubeFedConfig(t *testing.T) {
	errs := ValidateKubeFedConfig(validKubeFedConfig())
	if len(errs) != 0 {
		t.Errorf("expected success: %v", errs)
	}

	errorCases := map[string]*v1beta1.KubeFedConfig{}

	validScope := validKubeFedConfig()
	validScope.Spec.Scope = "NeitherClusterOrNamespaceScoped"
	errorCases["spec.scope: Unsupported value"] = validScope

	validAvailableDelayGreaterThan0 := validKubeFedConfig()
	validAvailableDelayGreaterThan0.Spec.ControllerDuration.AvailableDelay.Duration = 0
	errorCases["spec.controllerDuration.availableDelay: Invalid value"] = validAvailableDelayGreaterThan0

	validUnavailableDelayGreaterThan0 := validKubeFedConfig()
	validUnavailableDelayGreaterThan0.Spec.ControllerDuration.UnavailableDelay.Duration = 0
	errorCases["spec.controllerDuration.unavailableDelay: Invalid value"] = validUnavailableDelayGreaterThan0

	validLeaseDurationGreaterThan0 := validKubeFedConfig()
	validLeaseDurationGreaterThan0.Spec.LeaderElect.LeaseDuration.Duration = 0
	errorCases["spec.leaderElect.leaseDuration: Invalid value"] = validLeaseDurationGreaterThan0

	validRenewDeadlineGreaterThan0 := validKubeFedConfig()
	validRenewDeadlineGreaterThan0.Spec.LeaderElect.RenewDeadline.Duration = 0
	errorCases["spec.leaderElect.renewDeadline: Invalid value"] = validRenewDeadlineGreaterThan0

	// spec.leaderElect.leaderDuration must be greater than renewDeadline
	validElectorLeaseDurationGreater := validKubeFedConfig()
	validElectorLeaseDurationGreater.Spec.LeaderElect.LeaseDuration.Duration = 1
	validElectorLeaseDurationGreater.Spec.LeaderElect.RenewDeadline.Duration = 2
	errorCases["spec.leaderElect.leaseDuration: Invalid value"] = validElectorLeaseDurationGreater

	validRetryPeriodGreaterThan0 := validKubeFedConfig()
	validRetryPeriodGreaterThan0.Spec.LeaderElect.RetryPeriod.Duration = 0
	errorCases["spec.leaderElect.retryPeriod: Invalid value"] = validRetryPeriodGreaterThan0

	// spec.leaderElect.renewDeadline must be greater than retryPeriod*JitterFactor(1.2)
	validElectorDuration := validKubeFedConfig()
	validElectorDuration.Spec.LeaderElect.RenewDeadline.Duration = 12
	validElectorDuration.Spec.LeaderElect.RetryPeriod.Duration = 10
	errorCases["spec.leaderElect.renewDeadline: Invalid value"] = validElectorDuration

	validElectorResourceLock := validKubeFedConfig()
	invalidElectorResourceLock := v1beta1.ResourceLockType("NeitherConfigmapsOrEndpoints")
	validElectorResourceLock.Spec.LeaderElect.ResourceLock = &invalidElectorResourceLock
	errorCases["spec.leaderElect.resourceLock: Unsupported value"] = validElectorResourceLock

	validFeatureGateName := validKubeFedConfig()
	validFeatureGateName.Spec.FeatureGates[0].Name = "BadFeatureName"
	errorCases["spec.featureGates.name: Unsupported value"] = validFeatureGateName

	validDupFeatureGates := validKubeFedConfig()
	dupFeature := v1beta1.FeatureGatesConfig{
		Name:          string(features.PushReconciler),
		Configuration: v1beta1.ConfigurationEnabled,
	}
	validDupFeatureGates.Spec.FeatureGates = append(validDupFeatureGates.Spec.FeatureGates, dupFeature)
	errorCases["spec.featureGates.name: Duplicate value"] = validDupFeatureGates

	validFeatureGateConf := validKubeFedConfig()
	validFeatureGateConf.Spec.FeatureGates[0].Configuration = v1beta1.ConfigurationMode("NeitherEnableOrDisable")
	errorCases["spec.featureGates.configuration: Unsupported value"] = validFeatureGateConf

	zeroInt := int64(0)
	zeroIntPtr := &zeroInt

	validPeriodSecondsGreaterThan0 := validKubeFedConfig()
	validPeriodSecondsGreaterThan0.Spec.ClusterHealthCheck.PeriodSeconds = zeroIntPtr
	errorCases["spec.clusterHealthCheck.periodSeconds: Invalid value"] = validPeriodSecondsGreaterThan0

	validFailureThresholdGreaterThan0 := validKubeFedConfig()
	validFailureThresholdGreaterThan0.Spec.ClusterHealthCheck.FailureThreshold = zeroIntPtr
	errorCases["spec.clusterHealthCheck.failureThreshold: Invalid value"] = validFailureThresholdGreaterThan0

	validSuccessThresholdGreaterThan0 := validKubeFedConfig()
	validSuccessThresholdGreaterThan0.Spec.ClusterHealthCheck.SuccessThreshold = zeroIntPtr
	errorCases["spec.clusterHealthCheck.successThreshold: Invalid value"] = validSuccessThresholdGreaterThan0

	validTimeoutSecondsGreaterThan0 := validKubeFedConfig()
	validTimeoutSecondsGreaterThan0.Spec.ClusterHealthCheck.TimeoutSeconds = zeroIntPtr
	errorCases["spec.clusterHealthCheck.timeoutSeconds: Invalid value"] = validTimeoutSecondsGreaterThan0

	validAdoptResources := validKubeFedConfig()
	invalidAdoptResourcesValue := v1beta1.ResourceAdoption("NeitherEnableOrDisable")
	validAdoptResources.Spec.SyncController.AdoptResources = &invalidAdoptResourcesValue
	errorCases["spec.syncController.adoptResources: Unsupported value"] = validAdoptResources

	for k, v := range errorCases {
		errs := ValidateKubeFedConfig(v)
		if len(errs) == 0 {
			t.Errorf("[%s] expected failure", k)
		} else if !strings.Contains(errs[0].Error(), k) {
			t.Errorf("unexpected error: %q, expected: %q", errs[0].Error(), k)
		}
	}
}

func validKubeFedConfig() *v1beta1.KubeFedConfig {
	kfc := &v1beta1.KubeFedConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: util.DefaultKubeFedSystemNamespace,
			Name:      util.KubeFedConfigName,
		},
		Spec: v1beta1.KubeFedConfigSpec{
			Scope: apiextv1b1.ClusterScoped,
		},
	}

	app.SetDefaultKubeFedConfig(kfc)

	for _, name := range features.FeatureNames {
		feature := v1beta1.FeatureGatesConfig{Name: string(name), Configuration: v1beta1.ConfigurationEnabled}
		kfc.Spec.FeatureGates = append(kfc.Spec.FeatureGates, feature)
	}

	return kfc
}
