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
	"net/http"
	"sync"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/klog"

	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1/validation"
)

type KubeFedClusterValidationHook struct {
	client dynamic.ResourceInterface

	lock        sync.RWMutex
	initialized bool
}

func (a *KubeFedClusterValidationHook) ValidatingResource() (plural schema.GroupVersionResource, singular string) {
	return schema.GroupVersionResource{
			Group:    "admission.core.kubefed.k8s.io",
			Version:  "v1beta1",
			Resource: "kubefedclusters",
		},
		"kubefedcluster"
}

func (a *KubeFedClusterValidationHook) Validate(admissionSpec *admissionv1beta1.AdmissionRequest) *admissionv1beta1.AdmissionResponse {
	status := &admissionv1beta1.AdmissionResponse{}

	// We want to let through:
	// - Requests that are not for create, update
	// - Requests for subresources
	// - Requests for things that are not kubefedclusters
	if (admissionSpec.Operation != admissionv1beta1.Create && admissionSpec.Operation != admissionv1beta1.Update) ||
		len(admissionSpec.SubResource) != 0 ||
		(admissionSpec.Resource.Group != "core.kubefed.k8s.io" && admissionSpec.Resource.Resource != "kubefedclusters") {
		status.Allowed = true
		return status
	}

	klog.V(4).Infof("Validating AdmissionRequest = %v", admissionSpec)

	admittingObject := &v1beta1.KubeFedCluster{}
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

	errs := validation.ValidateKubeFedCluster(admittingObject)
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

func (a *KubeFedClusterValidationHook) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.initialized = true

	shallowClientConfigCopy := *kubeClientConfig
	shallowClientConfigCopy.GroupVersion = &schema.GroupVersion{
		Group:   "core.kubefed.k8s.io",
		Version: "v1beta1",
	}
	shallowClientConfigCopy.APIPath = "/apis"
	dynamicClient, err := dynamic.NewForConfig(&shallowClientConfigCopy)
	if err != nil {
		return err
	}
	a.client = dynamicClient.Resource(schema.GroupVersionResource{
		Group:    "core.kubefed.k8s.io",
		Version:  "v1beta1",
		Resource: "KubeFedCluster",
	})

	return nil
}
