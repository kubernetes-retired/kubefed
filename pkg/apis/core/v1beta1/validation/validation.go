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
	"strings"

	apiextv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apimachineryval "k8s.io/apimachinery/pkg/api/validation"
	valutil "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"sigs.k8s.io/kubefed/pkg/apis/core/typeconfig"
	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
)

func ValidateFederatedTypeConfig(obj *v1beta1.FederatedTypeConfig, statusSubResource bool) field.ErrorList {
	var allErrs field.ErrorList
	if !statusSubResource {
		allErrs = ValidateFederatedTypeConfigName(obj)
		allErrs = append(allErrs, ValidateFederatedTypeConfigSpec(&obj.Spec, field.NewPath("spec"))...)
	} else {
		allErrs = ValidateFederatedTypeConfigStatus(&obj.Status, field.NewPath("status"))
	}
	return allErrs
}

const federatedTypeConfigNameErrorMsg string = "name must be 'TARGET_PLURAL_NAME(.TARGET_GROUP_NAME)'"

func ValidateFederatedTypeConfigName(obj *v1beta1.FederatedTypeConfig) field.ErrorList {
	expectedName := typeconfig.GroupQualifiedName(obj.GetTargetType())
	if expectedName != obj.Name {
		return field.ErrorList{field.Invalid(field.NewPath("name"), obj.Name, federatedTypeConfigNameErrorMsg)}
	}
	return field.ErrorList{}
}

func ValidateFederatedTypeConfigSpec(spec *v1beta1.FederatedTypeConfigSpec, fldPath *field.Path) field.ErrorList {
	allErrs := ValidateAPIResource(&spec.TargetType, fldPath.Child("targetType"))
	allErrs = append(allErrs, validateEnumStrings(fldPath.Child("propagation"), string(spec.Propagation), []string{string(v1beta1.PropagationEnabled), string(v1beta1.PropagationDisabled)})...)
	allErrs = append(allErrs, ValidateFederatedAPIResource(&spec.FederatedType, fldPath.Child("federatedType"))...)
	if spec.StatusType != nil {
		allErrs = append(allErrs, ValidateStatusAPIResource(spec.StatusType, fldPath.Child("statusType"))...)
	}

	if spec.StatusCollection != nil {
		allErrs = append(allErrs, validateEnumStrings(fldPath.Child("statusCollection"), string(*spec.StatusCollection), []string{string(v1beta1.StatusCollectionEnabled), string(v1beta1.StatusCollectionDisabled)})...)
	}

	return allErrs
}

const domainWithAtLeastOneDot string = "should be a domain with at least one dot"

func ValidateFederatedAPIResource(fedType *v1beta1.APIResource, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(fedType.Group) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("group"), ""))
	} else if len(strings.Split(fedType.Group, ".")) < 2 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("group"), fedType.Group, domainWithAtLeastOneDot))
	}

	allErrs = append(allErrs, ValidateAPIResource(fedType, fldPath)...)
	return allErrs
}

func ValidateStatusAPIResource(statusType *v1beta1.APIResource, fldPath *field.Path) field.ErrorList {
	return ValidateFederatedAPIResource(statusType, fldPath)
}

func ValidateAPIResource(obj *v1beta1.APIResource, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(obj.Group) != 0 {
		if errs := valutil.IsDNS1123Subdomain(obj.Group); len(errs) > 0 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("group"), obj.Group, strings.Join(errs, ",")))
		}
	}

	if len(obj.Version) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("version"), ""))
	} else if errs := valutil.IsDNS1035Label(obj.Version); len(errs) > 0 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("version"), obj.Version, strings.Join(errs, ",")))
	}

	if len(obj.Kind) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("kind"), ""))
	} else if errs := valutil.IsDNS1035Label(strings.ToLower(obj.Kind)); len(errs) > 0 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("kind"), obj.Kind, strings.Join(errs, ",")))
	}

	if len(obj.PluralName) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("pluralName"), ""))
	} else if errs := valutil.IsDNS1035Label(obj.PluralName); len(errs) > 0 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("pluralName"), obj.PluralName, strings.Join(errs, ",")))
	}

	allErrs = append(allErrs, validateEnumStrings(fldPath.Child("scope"), string(obj.Scope), []string{string(apiextv1b1.ClusterScoped), string(apiextv1b1.NamespaceScoped)})...)

	return allErrs
}

func validateEnumStrings(fldPath *field.Path, value string, accepted []string) field.ErrorList {
	if value == "" {
		return field.ErrorList{field.Required(fldPath, "")}
	}
	for _, a := range accepted {
		if a == value {
			return field.ErrorList{}
		}
	}
	return field.ErrorList{field.NotSupported(fldPath, value, accepted)}
}

func ValidateFederatedTypeConfigStatus(status *v1beta1.FederatedTypeConfigStatus, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, apimachineryval.ValidateNonnegativeField(status.ObservedGeneration, fldPath.Child("observedGeneration"))...)
	allErrs = append(allErrs, validateEnumStrings(fldPath.Child("propagationController"), string(status.PropagationController), []string{string(v1beta1.ControllerStatusRunning), string(v1beta1.ControllerStatusNotRunning)})...)

	if status.StatusController != nil {
		allErrs = append(allErrs, validateEnumStrings(fldPath.Child("statusController"), string(*status.StatusController), []string{string(v1beta1.ControllerStatusRunning), string(v1beta1.ControllerStatusNotRunning)})...)
	}
	return allErrs
}

func ValidateKubeFedCluster(object *v1beta1.KubeFedCluster) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}
