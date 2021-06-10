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
	"context"
	"encoding/json"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1/validation"
	"sigs.k8s.io/kubefed/pkg/controller/webhook"
)

const (
	ResourceName       = "FederatedTypeConfig"
	resourcePluralName = "federatedtypeconfigs"
)

type FederatedTypeConfigAdmissionHook struct{}

var _ admission.Handler = &FederatedTypeConfigAdmissionHook{}

func (a *FederatedTypeConfigAdmissionHook) Handle(ctx context.Context, admissionSpec admission.Request) admission.Response {
	klog.V(4).Infof("Validating %q AdmissionRequest = %s", ResourceName, webhook.AdmissionRequestDebugString(admissionSpec))

	// We want to let through:
	// - Requests that are not for create, update
	// - Requests for things that are not FederatedTypeConfigs
	if webhook.Allowed(admissionSpec, resourcePluralName) {
		return admission.Response{
			AdmissionResponse: admissionv1.AdmissionResponse{
				Allowed: true,
			},
		}
	}

	admittingObject := &v1beta1.FederatedTypeConfig{}
	if err := json.Unmarshal(admissionSpec.Object.Raw, admittingObject); err != nil {
		return admission.Response{
			AdmissionResponse: admissionv1.AdmissionResponse{
				Allowed: false,
				Result: &metav1.Status{
					Status: metav1.StatusFailure, Code: http.StatusBadRequest, Reason: metav1.StatusReasonBadRequest,
					Message: err.Error(),
				},
			},
		}
	}

	klog.V(4).Infof("Validating %q = %+v", ResourceName, *admittingObject)

	isStatusSubResource := len(admissionSpec.SubResource) != 0
	return webhook.Validate(func() field.ErrorList {
		return validation.ValidateFederatedTypeConfig(admittingObject, isStatusSubResource)
	})
}
