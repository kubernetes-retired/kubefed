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

package webhook

import (
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
)

var (
	validationGroup   = "admission." + v1beta1.SchemeGroupVersion.Group
	validationVersion = v1beta1.SchemeGroupVersion.Version
)

func NewValidatingResource(resourcePluralName string) schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    validationGroup,
		Version:  validationVersion,
		Resource: resourcePluralName,
	}
}

// Allowed returns true if the admission request for the plural name of the
// resource passed in should be allowed to pass through, false otherwise.
func Allowed(a *admissionv1beta1.AdmissionRequest, pluralResourceName string) bool {
	// We want to let through:
	// - Requests that are not for create, update
	// - Requests for things that are not <pluralResourceName>
	createOrUpdate := a.Operation == admissionv1beta1.Create || a.Operation == admissionv1beta1.Update
	isMyGroupAndResource := a.Resource.Group == v1beta1.SchemeGroupVersion.Group && a.Resource.Resource == pluralResourceName
	return !createOrUpdate || !isMyGroupAndResource
}
