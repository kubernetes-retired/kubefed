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
	"k8s.io/apimachinery/pkg/util/validation/field"

	"sigs.k8s.io/kubefed/pkg/apis/core/v1alpha1"
)

func ValidateFederatedTypeConfig(object *v1alpha1.FederatedTypeConfig) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, ValidateAPIResource(&object.Spec.Target, field.NewPath("spec", "target"))...)

	return allErrs
}

func ValidateKubefedCluster(object *v1alpha1.KubefedCluster) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

func ValidateAPIResource(object *v1alpha1.APIResource, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(object.Kind) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("kind"), "kind is required"))
	}

	return allErrs
}
