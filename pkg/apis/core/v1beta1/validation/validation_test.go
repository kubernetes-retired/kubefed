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
	"time"

	corev1 "k8s.io/api/core/v1"
	apiextv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"sigs.k8s.io/kubefed/pkg/apis/core/common"
	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1/defaults"
	"sigs.k8s.io/kubefed/pkg/controller/util"
	"sigs.k8s.io/kubefed/pkg/features"
	"sigs.k8s.io/kubefed/pkg/kubefedctl/enable"
	"sigs.k8s.io/kubefed/pkg/kubefedctl/options"
	testcommon "sigs.k8s.io/kubefed/test/common"
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
	for _, successCase := range successCasesForFederatedTypeConfig() {
		if errs := ValidateFederatedTypeConfigName(successCase); len(errs) != 0 {
			t.Errorf("expected success: %v", errs)
		}
	}

	errorCases := map[string]*v1beta1.FederatedTypeConfig{}

	invalidDeploymentFedTypeConfig := federatedTypeConfig(apiResourceWithNonEmptyGroup())
	invalidDeploymentFedTypeConfig.Name = "deployments"
	errorCases[federatedTypeConfigNameErrorMsg] = invalidDeploymentFedTypeConfig

	invalidServicesFedTypeConfig := federatedTypeConfig(apiResourceWithEmptyGroup())
	invalidServicesFedTypeConfig.Name = "service"
	errorCases["name must be 'TARGET_PLURAL_NAME"] = invalidServicesFedTypeConfig

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
	for _, successCase := range successCasesForFederatedTypeConfig() {
		if errs := ValidateFederatedTypeConfigSpec(&successCase.Spec, field.NewPath("spec")); len(errs) != 0 {
			t.Errorf("expected success: %v", errs)
		}
	}

	errorCases := map[string]*v1beta1.FederatedTypeConfig{}

	// Validate required fields

	fedGroupRequired := validFederatedTypeConfig()
	fedGroupRequired.Spec.FederatedType.Group = ""
	errorCases["federatedType.group: Required value"] = fedGroupRequired

	statusGroupRequired := validFederatedTypeConfig()
	statusGroupRequired.Spec.StatusType.Group = ""
	errorCases["statusType.group: Required value"] = statusGroupRequired

	propagation := validFederatedTypeConfig()
	propagation.Spec.Propagation = ""
	errorCases["spec.propagation: Required value"] = propagation

	// Validate field values

	// Test only single error condition for each <Target|Federated|Status>Type
	// field.
	invalidFedGroup := validFederatedTypeConfig()
	invalidFedGroup.Spec.FederatedType.Group = "nodomain"
	errorCases[domainWithAtLeastOneDot] = invalidFedGroup

	invalidVersion := validFederatedTypeConfig()
	invalidVersion.Spec.TargetType.Version = "Alpha"
	errorCases["must consist of lower case alphanumeric characters"] = invalidVersion

	invalidPluralName := validFederatedTypeConfig()
	invalidPluralName.Spec.StatusType.PluralName = "2InvalidKind"
	errorCases["start with an alphabetic character"] = invalidPluralName

	invalidPropagation := validFederatedTypeConfig()
	invalidPropagation.Spec.Propagation = "InvalidPropagationMode"
	errorCases["spec.propagation: Unsupported value"] = invalidPropagation

	invalidStatusCollection := validFederatedTypeConfig()
	var invalidStatusCollectionMode v1beta1.StatusCollectionMode = "InvalidStatusCollectionMode"
	invalidStatusCollection.Spec.StatusCollection = &invalidStatusCollectionMode
	errorCases["spec.statusCollection: Unsupported value"] = invalidStatusCollection

	for k, v := range errorCases {
		errs := ValidateFederatedTypeConfigSpec(&v.Spec, field.NewPath("spec"))
		if len(errs) == 0 {
			t.Errorf("[%s] expected failure", k)
		} else if !strings.Contains(errs[0].Error(), k) {
			t.Errorf("unexpected error: %q, expected: %q", errs[0].Error(), k)
		}
	}
}

func TestValidateAPIResource(t *testing.T) {
	for _, successCase := range successCasesForAPIResource() {
		if errs := ValidateAPIResource(successCase, field.NewPath(".")); len(errs) != 0 {
			t.Errorf("expected success: %v", errs)
		}
	}

	errorCases := map[string]*v1beta1.APIResource{}

	// Validate required fields
	versionRequired := validAPIResource()
	versionRequired.Version = ""
	errorCases["version: Required value"] = versionRequired

	kindRequired := validAPIResource()
	kindRequired.Kind = ""
	errorCases["kind: Required value"] = kindRequired

	pluralNameRequired := validAPIResource()
	pluralNameRequired.PluralName = ""
	errorCases["pluralName: Required value"] = pluralNameRequired

	scopeRequired := validAPIResource()
	scopeRequired.Scope = ""
	errorCases["scope: Required value"] = scopeRequired

	// Validate field values
	invalidGroup := validAPIResource()
	invalidGroup.Group = "invalid#group"
	errorCases["consist of lower case alphanumeric characters, '-' or '.'"] = invalidGroup

	invalidVersion := validAPIResource()
	invalidVersion.Version = "Alpha"
	errorCases["must consist of lower case alphanumeric characters"] = invalidVersion

	invalidKind := validAPIResource()
	invalidKind.Kind = "Invalid.Kind"
	errorCases["alphanumeric characters or '-'"] = invalidKind

	invalidPluralName := validAPIResource()
	invalidPluralName.PluralName = "2InvalidKind"
	errorCases["start with an alphabetic character"] = invalidPluralName

	invalidScope := validAPIResource()
	invalidScope.Scope = "NeitherClusterOrNamespaceScoped"
	errorCases["scope: Unsupported value"] = invalidScope

	for k, v := range errorCases {
		errs := ValidateAPIResource(v, field.NewPath("."))
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

func successCasesForFederatedTypeConfig() []*v1beta1.FederatedTypeConfig {
	return []*v1beta1.FederatedTypeConfig{
		federatedTypeConfig(apiResourceWithEmptyGroup()),
		federatedTypeConfig(apiResourceWithNonEmptyGroup()),
	}
}

func successCasesForAPIResource() []*v1beta1.APIResource {
	return []*v1beta1.APIResource{
		apiResource(apiResourceWithEmptyGroup()),
		apiResource(apiResourceWithNonEmptyGroup()),
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
	enableTypeDirective := enable.NewEnableTypeDirective()
	typeConfig := enable.GenerateTypeConfigForTarget(*apiResource, enableTypeDirective)
	ftc := typeConfig.(*v1beta1.FederatedTypeConfig)

	kind := apiResource.Kind
	pluralName := apiResource.Name
	statusCollection := v1beta1.StatusCollectionEnabled
	statusController := v1beta1.ControllerStatusNotRunning
	ftc.Spec.StatusType = &v1beta1.APIResource{
		Group:      options.DefaultFederatedGroup,
		Version:    options.DefaultFederatedVersion,
		Kind:       fmt.Sprintf("Federated%sStatus", kind),
		PluralName: fmt.Sprintf("federated%sstatus", pluralName),
		Scope:      enable.FederatedNamespacedToScope(*apiResource),
	}
	ftc.Spec.StatusCollection = &statusCollection
	ftc.Status = v1beta1.FederatedTypeConfigStatus{
		ObservedGeneration:    1,
		PropagationController: v1beta1.ControllerStatusRunning,
		StatusController:      &statusController,
	}
	return ftc
}

func validAPIResource() *v1beta1.APIResource {
	return apiResource(apiResourceWithNonEmptyGroup())
}

func apiResource(apiResource *metav1.APIResource) *v1beta1.APIResource {
	return &v1beta1.APIResource{
		Group:      options.DefaultFederatedGroup,
		Version:    options.DefaultFederatedVersion,
		Kind:       fmt.Sprintf("Federated%s", apiResource.Kind),
		PluralName: fmt.Sprintf("federated%s", apiResource.Name),
		Scope:      enable.FederatedNamespacedToScope(*apiResource),
	}
}

func TestValidateKubeFedCluster(t *testing.T) {
	// Validate single success case for spec and status to ensure validation
	// functions are wired correctly.
	statusSubResource := []bool{true, false}
	validKFC := testcommon.ValidKubeFedCluster()
	for _, status := range statusSubResource {
		if errs := ValidateKubeFedCluster(validKFC, status); len(errs) != 0 {
			t.Errorf("expected success: %v", errs)
		}
	}

	// Validate single error case for spec and status to ensure validation
	// functions are wired correctly.
	type KFCAndStatusSubResource struct {
		kfc    *v1beta1.KubeFedCluster
		status bool
	}
	errorCases := map[string]KFCAndStatusSubResource{}

	invalidKFCSpec := testcommon.ValidKubeFedCluster()
	invalidKFCSpec.Spec.APIEndpoint = ""
	errorCases["apiEndpoint: Required value"] = KFCAndStatusSubResource{
		invalidKFCSpec,
		false,
	}

	invalidKFCStatus := testcommon.ValidKubeFedCluster()
	invalidKFCStatus.Status.Conditions[1].Type = ""
	errorCases["conditions[1].type: Required value"] = KFCAndStatusSubResource{
		invalidKFCStatus,
		true,
	}

	for k, v := range errorCases {
		errs := ValidateKubeFedCluster(v.kfc, v.status)
		if len(errs) == 0 {
			t.Errorf("[%s] expected failure", k)
		} else if !strings.Contains(errs[0].Error(), k) {
			t.Errorf("unexpected error: %q, expected: %q", errs[0].Error(), k)
		}
	}
}

func TestValidateAPIEndpoint(t *testing.T) {
	successProtocolSchemes := []string{
		"",
		"http://",
		"https://",
	}
	successAPIEndpoints := []string{
		"example.com",
		"my.example.com",
		"192.0.2.219",
		"[2001:db8:25a4:8d2::1]",
	}
	successPortNums := []string{
		"",
		":1",
		":80",
		":8080",
		":65535",
	}
	successURLPath := []string{
		"",
		"/",
		"/path/to/endpoint",
	}

	for _, scheme := range successProtocolSchemes {
		for _, endpt := range successAPIEndpoints {
			for _, port := range successPortNums {
				for _, path := range successURLPath {
					if scheme == "" && path == "/path/to/endpoint" {
						// An empty protocol scheme with a URL path (e.g.
						// /path/to/endpoint) is not supported so skip it.
						continue
					}
					errs := validateAPIEndpoint(scheme+endpt+port+path, field.NewPath("apiEndpoint"))
					if len(errs) != 0 {
						t.Errorf("expected success: %v", errs)
					}
				}
			}
		}
	}

	errorCases := []struct {
		apiEndpoint    string
		expectedErrMsg string
	}{
		{
			"",
			"apiEndpoint: Required value",
		},
		{
			"https://",
			`apiEndpoint: Invalid value: "https://": host must be a URL or a host:port pair`,
		},
		{
			"example.com/path/to/somewhere",
			`apiEndpoint: Invalid value: "example.com/path/to/somewhere": host must be a URL or a host:port pair`,
		},
		{
			"192.0.2.35/path/to/somewhere",
			`apiEndpoint: Invalid value: "192.0.2.35/path/to/somewhere": host must be a URL or a host:port pair`,
		},
		{
			"tcp://example.com",
			`apiEndpoint: Unsupported value: "tcp"`,
		},
		{
			"example_com",
			"lower case alphanumeric characters, '-' or '.'",
		},
		{
			"-example.com",
			"must start and end with an alphanumeric character",
		},
		{
			"192.0.2..161",
			`apiEndpoint: Invalid value: "192.0.2..161": must be a valid IP address`,
		},
		{
			"[2001:db8:25a4::8d2::1]",
			`apiEndpoint: Invalid value: "2001:db8:25a4::8d2::1": must be a valid IP address`,
		},
		{
			"example.com:port80",
			`apiEndpoint: Invalid value: "port80": error converting port to integer`,
		},
		{
			"example.com:-80",
			"apiEndpoint: Invalid value: -80: must be between 1 and 65535, inclusive",
		},
		{
			"example.com:0",
			"apiEndpoint: Invalid value: 0: must be between 1 and 65535, inclusive",
		},
		{
			"example.com:65536",
			"apiEndpoint: Invalid value: 65536: must be between 1 and 65535, inclusive",
		},
	}

	for _, test := range errorCases {
		errs := validateAPIEndpoint(test.apiEndpoint, field.NewPath("apiEndpoint"))
		if len(errs) == 0 {
			t.Errorf("[%s] expected failure", test.expectedErrMsg)
		} else {
			matchedErr := false
			for _, err := range errs {
				if strings.Contains(err.Error(), test.expectedErrMsg) {
					matchedErr = true
					break
				}
			}
			if !matchedErr {
				t.Errorf("unexpected error: %v, expected: %q", errs, test.expectedErrMsg)
			}
		}
	}
}

func TestValidateLocalSecretReference(t *testing.T) {
	testCases := []struct {
		secretName     string
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			"validation-test-cluster1",
			false,
			"",
		},
		{
			"",
			true,
			"name: Required value",
		},
		{
			"invalid_secretname",
			true,
			"must consist of lower case alphanumeric characters, '-' or '.'",
		},
	}

	for _, test := range testCases {
		secretRef := &v1beta1.LocalSecretReference{
			Name: test.secretName,
		}
		errs := validateLocalSecretReference(secretRef, field.NewPath("secretRef"))
		hasErr := len(errs) > 0
		if hasErr && hasErr != test.expectedErr {
			t.Errorf("[%s] expected failure", test.expectedErrMsg)
		} else if hasErr && !strings.Contains(errs[0].Error(), test.expectedErrMsg) {
			t.Errorf("unexpected error: %v, expected: %q", errs[0].Error(), test.expectedErrMsg)
		}
	}
}

func TestDisabledTLSValidations(t *testing.T) {
	testCases := []struct {
		disabledTLSValidations []v1beta1.TLSValidation
		expectedErr            bool
		expectedErrMsg         string
	}{
		{
			[]v1beta1.TLSValidation{},
			false,
			"",
		},
		{
			[]v1beta1.TLSValidation{v1beta1.TLSAll},
			false,
			"",
		},
		{
			[]v1beta1.TLSValidation{v1beta1.TLSSubjectName},
			false,
			"",
		},
		{
			[]v1beta1.TLSValidation{v1beta1.TLSValidityPeriod},
			false,
			"",
		},
		{
			[]v1beta1.TLSValidation{v1beta1.TLSSubjectName, v1beta1.TLSAll},
			true,
			"when * is specified, it is expected to be the only option in list",
		},
		{
			[]v1beta1.TLSValidation{v1beta1.TLSAll, v1beta1.TLSValidityPeriod},
			true,
			"when * is specified, it is expected to be the only option in list",
		},
	}

	for _, test := range testCases {
		errs := validateDisabledTLSValidations(test.disabledTLSValidations, field.NewPath("disabledTLSValidations"))
		hasErr := len(errs) > 0
		if hasErr && hasErr != test.expectedErr {
			t.Errorf("[%s] expected failure", test.expectedErrMsg)
		} else if hasErr && !strings.Contains(errs[0].Error(), test.expectedErrMsg) {
			t.Errorf("unexpected error: %v, expected: %q", errs[0].Error(), test.expectedErrMsg)
		}
	}
}

func TestValidateClusterCondition(t *testing.T) {
	testCases := []struct {
		cc             *v1beta1.ClusterCondition
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			cc: &v1beta1.ClusterCondition{
				Type:   common.ClusterReady,
				Status: corev1.ConditionTrue,
				LastProbeTime: metav1.Time{
					Time: time.Now(),
				},
			},
			expectedErr:    false,
			expectedErrMsg: "",
		},
		{
			cc: &v1beta1.ClusterCondition{
				Status: corev1.ConditionTrue,
				LastProbeTime: metav1.Time{
					Time: time.Now(),
				},
			},
			expectedErr:    true,
			expectedErrMsg: "conditions[0].type: Required value",
		},
		{
			cc: &v1beta1.ClusterCondition{
				Type: common.ClusterReady,
				LastProbeTime: metav1.Time{
					Time: time.Now(),
				},
			},
			expectedErr:    true,
			expectedErrMsg: "conditions[0].status: Required value",
		},
		{
			cc: &v1beta1.ClusterCondition{
				Type:   common.ClusterReady,
				Status: corev1.ConditionTrue,
			},
			expectedErr:    true,
			expectedErrMsg: "conditions[0].lastProbeTime: Required value",
		},
		{
			cc: &v1beta1.ClusterCondition{
				Type:   "Invalid",
				Status: corev1.ConditionTrue,
				LastProbeTime: metav1.Time{
					Time: time.Now(),
				},
			},
			expectedErr:    true,
			expectedErrMsg: "conditions[0].type: Unsupported value",
		},
		{
			cc: &v1beta1.ClusterCondition{
				Type:   common.ClusterReady,
				Status: "Invalid",
				LastProbeTime: metav1.Time{
					Time: time.Now(),
				},
			},
			expectedErr:    true,
			expectedErrMsg: "conditions[0].status: Unsupported value",
		},
	}

	for _, test := range testCases {
		errs := validateClusterCondition(test.cc, field.NewPath("conditions").Index(0))
		hasErr := len(errs) > 0
		if hasErr && hasErr != test.expectedErr {
			t.Errorf("[%s] expected failure", test.expectedErrMsg)
		} else if hasErr && !strings.Contains(errs[0].Error(), test.expectedErrMsg) {
			t.Errorf("unexpected error: %v, expected: %q", errs[0].Error(), test.expectedErrMsg)
		}
	}
}

func TestValidateKubeFedConfig(t *testing.T) {
	errs := ValidateKubeFedConfig(validKubeFedConfig())
	if len(errs) != 0 {
		t.Errorf("expected success: %v", errs)
	}

	errorCases := map[string]*v1beta1.KubeFedConfig{}

	invalidScope := validKubeFedConfig()
	invalidScope.Spec.Scope = "NeitherClusterOrNamespaceScoped"
	errorCases["spec.scope: Unsupported value"] = invalidScope

	invalidControllerDurationNil := validKubeFedConfig()
	invalidControllerDurationNil.Spec.ControllerDuration = nil
	errorCases["spec.controllerDuration: Required value"] = invalidControllerDurationNil

	invalidAvailableDelayNil := validKubeFedConfig()
	invalidAvailableDelayNil.Spec.ControllerDuration.AvailableDelay = nil
	errorCases["spec.controllerDuration.availableDelay: Required value"] = invalidAvailableDelayNil

	invalidAvailableDelayGreaterThan0 := validKubeFedConfig()
	invalidAvailableDelayGreaterThan0.Spec.ControllerDuration.AvailableDelay.Duration = 0
	errorCases["spec.controllerDuration.availableDelay: Invalid value"] = invalidAvailableDelayGreaterThan0

	invalidUnavailableDelayNil := validKubeFedConfig()
	invalidUnavailableDelayNil.Spec.ControllerDuration.UnavailableDelay = nil
	errorCases["spec.controllerDuration.unavailableDelay: Required value"] = invalidUnavailableDelayNil

	invalidUnavailableDelayGreaterThan0 := validKubeFedConfig()
	invalidUnavailableDelayGreaterThan0.Spec.ControllerDuration.UnavailableDelay.Duration = 0
	errorCases["spec.controllerDuration.unavailableDelay: Invalid value"] = invalidUnavailableDelayGreaterThan0

	invalidLeaderElectNil := validKubeFedConfig()
	invalidLeaderElectNil.Spec.LeaderElect = nil
	errorCases["spec.leaderElect: Required value"] = invalidLeaderElectNil

	invalidLeaseDurationNil := validKubeFedConfig()
	invalidLeaseDurationNil.Spec.LeaderElect.LeaseDuration = nil
	errorCases["spec.leaderElect.leaseDuration: Required value"] = invalidLeaseDurationNil

	invalidLeaseDurationGreaterThan0 := validKubeFedConfig()
	invalidLeaseDurationGreaterThan0.Spec.LeaderElect.LeaseDuration.Duration = 0
	errorCases["spec.leaderElect.leaseDuration: Invalid value"] = invalidLeaseDurationGreaterThan0

	invalidRenewDeadlineNil := validKubeFedConfig()
	invalidRenewDeadlineNil.Spec.LeaderElect.RenewDeadline = nil
	errorCases["spec.leaderElect.renewDeadline: Required value"] = invalidRenewDeadlineNil

	invalidRenewDeadlineGreaterThan0 := validKubeFedConfig()
	invalidRenewDeadlineGreaterThan0.Spec.LeaderElect.RenewDeadline.Duration = 0
	errorCases["spec.leaderElect.renewDeadline: Invalid value"] = invalidRenewDeadlineGreaterThan0

	// spec.leaderElect.leaderDuration must be greater than renewDeadline
	invalidElectorLeaseDurationGreater := validKubeFedConfig()
	invalidElectorLeaseDurationGreater.Spec.LeaderElect.LeaseDuration.Duration = 1
	invalidElectorLeaseDurationGreater.Spec.LeaderElect.RenewDeadline.Duration = 2
	errorCases["spec.leaderElect.leaseDuration: Invalid value"] = invalidElectorLeaseDurationGreater

	invalidRetryPeriodNil := validKubeFedConfig()
	invalidRetryPeriodNil.Spec.LeaderElect.RetryPeriod = nil
	errorCases["spec.leaderElect.retryPeriod: Required value"] = invalidRetryPeriodNil

	invalidRetryPeriodGreaterThan0 := validKubeFedConfig()
	invalidRetryPeriodGreaterThan0.Spec.LeaderElect.RetryPeriod.Duration = 0
	errorCases["spec.leaderElect.retryPeriod: Invalid value"] = invalidRetryPeriodGreaterThan0

	// spec.leaderElect.renewDeadline must be greater than retryPeriod*JitterFactor(1.2)
	invalidElectorDuration := validKubeFedConfig()
	invalidElectorDuration.Spec.LeaderElect.RenewDeadline.Duration = 12
	invalidElectorDuration.Spec.LeaderElect.RetryPeriod.Duration = 10
	errorCases["spec.leaderElect.renewDeadline: Invalid value"] = invalidElectorDuration

	invalidElectorResourceLockNil := validKubeFedConfig()
	invalidElectorResourceLockNil.Spec.LeaderElect.ResourceLock = nil
	errorCases["spec.leaderElect.resourceLock: Required value"] = invalidElectorResourceLockNil

	invalidElectorResourceLock := validKubeFedConfig()
	invalidElectorResourceLockType := v1beta1.ResourceLockType("NeitherConfigmapsOrEndpoints")
	invalidElectorResourceLock.Spec.LeaderElect.ResourceLock = &invalidElectorResourceLockType
	errorCases["spec.leaderElect.resourceLock: Unsupported value"] = invalidElectorResourceLock

	invalidFeatureGateNil := validKubeFedConfig()
	invalidFeatureGateNil.Spec.FeatureGates = nil
	errorCases["spec.featureGates: Required value"] = invalidFeatureGateNil

	invalidFeatureGateName := validKubeFedConfig()
	invalidFeatureGateName.Spec.FeatureGates[0].Name = "BadFeatureName"
	errorCases["spec.featureGates.name: Unsupported value"] = invalidFeatureGateName

	invalidDupFeatureGates := validKubeFedConfig()
	dupFeature := v1beta1.FeatureGatesConfig{
		Name:          string(features.PushReconciler),
		Configuration: v1beta1.ConfigurationEnabled,
	}
	invalidDupFeatureGates.Spec.FeatureGates = append(invalidDupFeatureGates.Spec.FeatureGates, dupFeature)
	errorCases["spec.featureGates.name: Duplicate value"] = invalidDupFeatureGates

	invalidFeatureGateConf := validKubeFedConfig()
	invalidFeatureGateConf.Spec.FeatureGates[0].Configuration = v1beta1.ConfigurationMode("NeitherEnableOrDisable")
	errorCases["spec.featureGates.configuration: Unsupported value"] = invalidFeatureGateConf

	invalidClusterHealthCheckNil := validKubeFedConfig()
	invalidClusterHealthCheckNil.Spec.ClusterHealthCheck = nil
	errorCases["spec.clusterHealthCheck: Required value"] = invalidClusterHealthCheckNil

	zeroInt := int64(0)
	zeroIntPtr := &zeroInt

	invalidPeriodNil := validKubeFedConfig()
	invalidPeriodNil.Spec.ClusterHealthCheck.Period = nil
	errorCases["spec.clusterHealthCheck.period: Required value"] = invalidPeriodNil

	invalidPeriodGreaterThan0 := validKubeFedConfig()
	invalidPeriodGreaterThan0.Spec.ClusterHealthCheck.Period.Duration = 0
	errorCases["spec.clusterHealthCheck.period: Invalid value"] = invalidPeriodGreaterThan0

	invalidFailureThresholdNil := validKubeFedConfig()
	invalidFailureThresholdNil.Spec.ClusterHealthCheck.FailureThreshold = nil
	errorCases["spec.clusterHealthCheck.failureThreshold: Required value"] = invalidFailureThresholdNil

	invalidFailureThresholdGreaterThan0 := validKubeFedConfig()
	invalidFailureThresholdGreaterThan0.Spec.ClusterHealthCheck.FailureThreshold = zeroIntPtr
	errorCases["spec.clusterHealthCheck.failureThreshold: Invalid value"] = invalidFailureThresholdGreaterThan0

	invalidSuccessThresholdNil := validKubeFedConfig()
	invalidSuccessThresholdNil.Spec.ClusterHealthCheck.SuccessThreshold = nil
	errorCases["spec.clusterHealthCheck.successThreshold: Required value"] = invalidSuccessThresholdNil

	invalidSuccessThresholdGreaterThan0 := validKubeFedConfig()
	invalidSuccessThresholdGreaterThan0.Spec.ClusterHealthCheck.SuccessThreshold = zeroIntPtr
	errorCases["spec.clusterHealthCheck.successThreshold: Invalid value"] = invalidSuccessThresholdGreaterThan0

	invalidTimeoutNil := validKubeFedConfig()
	invalidTimeoutNil.Spec.ClusterHealthCheck.Timeout = nil
	errorCases["spec.clusterHealthCheck.timeout: Required value"] = invalidTimeoutNil

	invalidTimeoutGreaterThan0 := validKubeFedConfig()
	invalidTimeoutGreaterThan0.Spec.ClusterHealthCheck.Timeout.Duration = 0
	errorCases["spec.clusterHealthCheck.timeout: Invalid value"] = invalidTimeoutGreaterThan0

	invalidSyncControllerNil := validKubeFedConfig()
	invalidSyncControllerNil.Spec.SyncController = nil
	errorCases["spec.syncController: Required value"] = invalidSyncControllerNil

	invalidAdoptResourcesNil := validKubeFedConfig()
	invalidAdoptResourcesNil.Spec.SyncController.AdoptResources = nil
	errorCases["spec.syncController.adoptResources: Required value"] = invalidAdoptResourcesNil

	invalidAdoptResources := validKubeFedConfig()
	invalidAdoptResourcesValue := v1beta1.ResourceAdoption("NeitherEnableOrDisable")
	invalidAdoptResources.Spec.SyncController.AdoptResources = &invalidAdoptResourcesValue
	errorCases["spec.syncController.adoptResources: Unsupported value"] = invalidAdoptResources

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

	defaults.SetDefaultKubeFedConfig(kfc)
	return kfc
}
