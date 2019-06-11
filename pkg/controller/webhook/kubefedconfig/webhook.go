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

package kubefedconfig

import (
	"strings"
	"sync"

	"github.com/openshift/generic-admission-server/pkg/apiserver"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/klog"

	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1/validation"
	"sigs.k8s.io/kubefed/pkg/controller/webhook"
)

const (
	resourceName       = "KubeFedConfig"
	resourcePluralName = "kubefedconfigs"
)

type KubeFedConfigAdmissionHook struct {
	client dynamic.ResourceInterface

	lock        sync.RWMutex
	initialized bool
}

var _ apiserver.ValidatingAdmissionHook = &KubeFedConfigAdmissionHook{}

func (a *KubeFedConfigAdmissionHook) ValidatingResource() (plural schema.GroupVersionResource, singular string) {
	klog.Infof("New ValidatingResource for %q", resourceName)
	return webhook.NewValidatingResource(resourcePluralName), strings.ToLower(resourceName)
}

func (a *KubeFedConfigAdmissionHook) Validate(admissionSpec *admissionv1beta1.AdmissionRequest) *admissionv1beta1.AdmissionResponse {
	status := &admissionv1beta1.AdmissionResponse{}

	klog.V(4).Infof("Validating %q AdmissionRequest = %s", resourceName, webhook.AdmissionRequestDebugString(admissionSpec))

	if webhook.Allowed(admissionSpec, resourcePluralName, status) {
		return status
	}

	admittingObject := &v1beta1.KubeFedConfig{}
	err := webhook.Unmarshal(admissionSpec, admittingObject, status)
	if err != nil {
		return status
	}

	if !webhook.Initialized(&a.initialized, &a.lock, status) {
		return status
	}

	klog.V(4).Infof("Validating %q = %+v", resourceName, *admittingObject)

	webhook.Validate(status, func() field.ErrorList {
		return validation.ValidateKubeFedConfig(admittingObject)
	})

	return status
}

func (a *KubeFedConfigAdmissionHook) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
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

	klog.Infof("Initialized admission webhook for %q", resourceName)
	return nil
}
