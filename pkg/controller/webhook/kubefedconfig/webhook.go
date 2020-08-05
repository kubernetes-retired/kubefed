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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1/defaults"
	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1/validation"
	"sigs.k8s.io/kubefed/pkg/controller/webhook"
)

const (
	ResourceName       = "KubeFedConfig"
	resourcePluralName = "kubefedconfigs"
)

type KubeFedConfigValidator struct{}

var _ admission.Handler = &KubeFedConfigValidator{}

func (a *KubeFedConfigValidator) Handle(ctx context.Context, admissionSpec admission.Request) admission.Response {
	status := admission.Response{}

	klog.V(4).Infof("Validating %q AdmissionRequest = %s", ResourceName, webhook.AdmissionRequestDebugString(admissionSpec))

	if webhook.Allowed(admissionSpec, resourcePluralName, &status) {
		return status
	}

	admittingObject := &v1beta1.KubeFedConfig{}
	err := webhook.Unmarshal(&admissionSpec.Object, admittingObject, &status)
	if err != nil {
		return status
	}

	var oldObject *v1beta1.KubeFedConfig
	if admissionSpec.Operation == admissionv1beta1.Update {
		oldObject = &v1beta1.KubeFedConfig{}
		err = webhook.Unmarshal(&admissionSpec.OldObject, oldObject, &status)
		if err != nil {
			return status
		}
	}

	klog.V(4).Infof("Validating %q = %+v", ResourceName, *admittingObject)

	webhook.Validate(&status, func() field.ErrorList {
		return validation.ValidateKubeFedConfig(admittingObject, oldObject)
	})

	return status
}

type KubeFedConfigDefaulter struct{}

var _ admission.Handler = &KubeFedConfigDefaulter{}

func (a *KubeFedConfigDefaulter) Handle(ctx context.Context, admissionSpec admission.Request) admission.Response {
	status := admission.Response{}
	klog.V(4).Infof("Admitting %q AdmissionRequest = %s", ResourceName, webhook.AdmissionRequestDebugString(admissionSpec))

	admittingObject := &v1beta1.KubeFedConfig{}
	err := webhook.Unmarshal(&admissionSpec.Object, admittingObject, &status)
	if err != nil {
		return status
	}

	klog.V(4).Infof("Admitting %q = %+v", ResourceName, *admittingObject)

	defaultedObject := admittingObject.DeepCopyObject().(*v1beta1.KubeFedConfig)
	defaults.SetDefaultKubeFedConfig(defaultedObject)

	if reflect.DeepEqual(admittingObject, defaultedObject) {
		status.Allowed = true
		return status
	}

	// TODO(font) Optimize by generalizing the ability to add only the fields
	// that have been defaulted. If merge patch is ever supported use that
	// instead.
	patchOperations := []patchOperation{
		{
			"replace",
			"/spec",
			defaultedObject.Spec,
		},
	}

	patchBytes, err := json.Marshal(patchOperations)
	if err != nil {
		status.Allowed = false
		status.Result = &metav1.Status{
			Status: metav1.StatusFailure, Code: http.StatusInternalServerError, Reason: metav1.StatusReasonInternalError,
			Message: fmt.Sprintf("Error marshalling defaulted KubeFedConfig json operation = %+v, err: %v", patchOperations, err),
		}
		return status
	}

	status.PatchType = new(admissionv1beta1.PatchType)
	*status.PatchType = admissionv1beta1.PatchTypeJSONPatch
	status.Patch = patchBytes
	status.Allowed = true
	return status
}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}
