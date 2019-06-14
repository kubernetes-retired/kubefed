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
	"sync"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/klog"

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
func Allowed(a *admissionv1beta1.AdmissionRequest, pluralResourceName string, status *admissionv1beta1.AdmissionResponse) bool {
	// We want to let through:
	// - Requests that are not for create, update
	// - Requests for things that are not <pluralResourceName>
	createOrUpdate := a.Operation == admissionv1beta1.Create || a.Operation == admissionv1beta1.Update
	isMyGroupAndResource := a.Resource.Group == v1beta1.SchemeGroupVersion.Group && a.Resource.Resource == pluralResourceName
	if !createOrUpdate || !isMyGroupAndResource {
		status.Allowed = true
		return true
	}
	return false
}

func Unmarshal(a *admissionv1beta1.AdmissionRequest, object interface{}, status *admissionv1beta1.AdmissionResponse) error {
	err := json.Unmarshal(a.Object.Raw, object)
	if err != nil {
		status.Allowed = false
		status.Result = &metav1.Status{
			Status: metav1.StatusFailure, Code: http.StatusBadRequest, Reason: metav1.StatusReasonBadRequest,
			Message: err.Error(),
		}
	}

	return err
}

func Initialized(initialized *bool, lock *sync.RWMutex, status *admissionv1beta1.AdmissionResponse) bool {
	lock.RLock()
	defer lock.RUnlock()

	if !*initialized {
		status.Allowed = false
		status.Result = &metav1.Status{
			Status: metav1.StatusFailure, Code: http.StatusInternalServerError, Reason: metav1.StatusReasonInternalError,
			Message: "not initialized",
		}
	}

	return *initialized
}

func Validate(status *admissionv1beta1.AdmissionResponse, validateFn func() field.ErrorList) {
	errs := validateFn()
	if len(errs) != 0 {
		status.Allowed = false
		status.Result = &metav1.Status{
			Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: metav1.StatusReasonForbidden,
			Message: errs.ToAggregate().Error(),
		}
	} else {
		status.Allowed = true
	}
}

func Initialize(kubeClientConfig *rest.Config, client *dynamic.ResourceInterface, lock *sync.RWMutex, initialized *bool, resourceName string) error {
	lock.Lock()
	defer lock.Unlock()

	*initialized = true

	shallowClientConfigCopy := *kubeClientConfig
	shallowClientConfigCopy.GroupVersion = &schema.GroupVersion{
		Group:   v1beta1.SchemeGroupVersion.Group,
		Version: v1beta1.SchemeGroupVersion.Version,
	}

	shallowClientConfigCopy.APIPath = "/apis"
	dynamicClient, err := dynamic.NewForConfig(&shallowClientConfigCopy)
	if err != nil {
		return err
	}

	*client = dynamicClient.Resource(schema.GroupVersionResource{
		Group:    v1beta1.SchemeGroupVersion.Group,
		Version:  v1beta1.SchemeGroupVersion.Version,
		Resource: resourceName,
	})

	klog.Infof("Initialized admission webhook for %q", resourceName)
	return nil
}

func AdmissionRequestDebugString(a *admissionv1beta1.AdmissionRequest) string {
	return fmt.Sprintf("UID=%v Kind={%v} Resource=%+v SubResource=%v Name=%v Namespace=%v Operation=%v UserInfo=%+v DryRun=%v",
		a.UID, a.Kind, a.Resource, a.SubResource, a.Name, a.Namespace, a.Operation, a.UserInfo, *a.DryRun)
}
