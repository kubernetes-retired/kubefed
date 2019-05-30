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

package federatedtypeconfig

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/klog"

	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1/validation"
	"sigs.k8s.io/kubefed/pkg/controller/webhook"
)

const (
	resourceName       = "FederatedTypeConfig"
	resourcePluralName = "federatedtypeconfigs"
)

type FederatedTypeConfigValidationHook struct {
	client dynamic.ResourceInterface

	lock        sync.RWMutex
	initialized bool
}

func (a *FederatedTypeConfigValidationHook) ValidatingResource() (plural schema.GroupVersionResource, singular string) {
	return webhook.NewValidatingResource(resourcePluralName), strings.ToLower(resourceName)
}

func (a *FederatedTypeConfigValidationHook) Validate(admissionSpec *admissionv1beta1.AdmissionRequest) *admissionv1beta1.AdmissionResponse {
	status := &admissionv1beta1.AdmissionResponse{}

	// We want to let through:
	// - Requests that are not for create, update
	// - Requests for things that are not FederatedTypeConfigs
	if webhook.Allowed(admissionSpec, resourcePluralName) {
		status.Allowed = true
		return status
	}

	klog.V(4).Infof("Validating AdmissionRequest = %v", admissionSpec)

	admittingObject := &v1beta1.FederatedTypeConfig{}
	err := json.Unmarshal(admissionSpec.Object.Raw, admittingObject)
	if err != nil {
		status.Allowed = false
		status.Result = &metav1.Status{
			Status: metav1.StatusFailure, Code: http.StatusBadRequest, Reason: metav1.StatusReasonBadRequest,
			Message: err.Error(),
		}
		return status
	}

	a.lock.RLock()
	defer a.lock.RUnlock()
	if !a.initialized {
		status.Allowed = false
		status.Result = &metav1.Status{
			Status: metav1.StatusFailure, Code: http.StatusInternalServerError, Reason: metav1.StatusReasonInternalError,
			Message: "not initialized",
		}
		return status
	}

	isStatusSubResource := len(admissionSpec.SubResource) != 0
	errs := validation.ValidateFederatedTypeConfig(admittingObject, isStatusSubResource)
	if len(errs) != 0 {
		status.Allowed = false
		status.Result = &metav1.Status{
			Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: metav1.StatusReasonForbidden,
			Message: errs.ToAggregate().Error(),
		}
		return status
	}

	status.Allowed = true
	return status
}

func (a *FederatedTypeConfigValidationHook) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.initialized = true

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
	a.client = dynamicClient.Resource(schema.GroupVersionResource{
		Group:    v1beta1.SchemeGroupVersion.Group,
		Version:  v1beta1.SchemeGroupVersion.Version,
		Resource: resourceName,
	})

	return nil
}
