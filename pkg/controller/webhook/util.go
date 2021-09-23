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
	"encoding/json"
	"fmt"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
)

var (
	validationGroup  = "validation." + v1beta1.SchemeGroupVersion.Group
	mutationGroup    = "mutation." + v1beta1.SchemeGroupVersion.Group
	admissionVersion = v1beta1.SchemeGroupVersion.Version
)

func NewMutatingResource(resourcePluralName string) schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    mutationGroup,
		Version:  admissionVersion,
		Resource: resourcePluralName,
	}
}

func NewValidatingResource(resourcePluralName string) schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    validationGroup,
		Version:  admissionVersion,
		Resource: resourcePluralName,
	}
}

// Allowed returns true if the admission request for the plural name of the
// resource passed in should be allowed to pass through, false otherwise.
func Allowed(a admission.Request, pluralResourceName string) bool {
	// We want to let through:
	// - Requests that are not for create, update
	// - Requests for things that are not <pluralResourceName>
	createOrUpdate := a.Operation == admissionv1.Create || a.Operation == admissionv1.Update
	isMyGroupAndResource := a.Resource.Group == v1beta1.SchemeGroupVersion.Group && a.Resource.Resource == pluralResourceName
	return !createOrUpdate || !isMyGroupAndResource
}

func Unmarshal(rawExt *runtime.RawExtension, object interface{}) error {
	return json.Unmarshal(rawExt.Raw, object)
}

func Validate(validateFn func() field.ErrorList) admission.Response {
	res := admission.Response{}
	errs := validateFn()
	if len(errs) != 0 {
		res.Allowed = false
		res.Result = &metav1.Status{
			Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: metav1.StatusReasonForbidden,
			Message: errs.ToAggregate().Error(),
		}
	} else {
		res.Allowed = true
	}

	return res
}

func AdmissionRequestDebugString(a admission.Request) string {
	return fmt.Sprintf("UID=%v Kind={%v} Resource=%+v SubResource=%v Name=%v Namespace=%v Operation=%v UserInfo=%+v DryRun=%v",
		a.UID, a.Kind, a.Resource, a.SubResource, a.Name, a.Namespace, a.Operation, a.UserInfo, *a.DryRun)
}
