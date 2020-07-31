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

package kubefedcluster

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/klog"

	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1/validation"
	"sigs.k8s.io/kubefed/pkg/controller/webhook"
)

const (
	ResourceName       = "KubeFedCluster"
	resourcePluralName = "kubefedclusters"
)

type KubeFedClusterAdmissionHook struct{}

var _ admission.Handler = &KubeFedClusterAdmissionHook{}

func (a *KubeFedClusterAdmissionHook) Handle(ctx context.Context, admissionSpec admission.Request) admission.Response {
	status := admission.Response{}

	klog.V(4).Infof("Validating %q AdmissionRequest = %s", ResourceName, webhook.AdmissionRequestDebugString(admissionSpec))

	// We want to let through:
	// - Requests that are not for create, update
	// - Requests for things that are not FederatedTypeConfigs
	if webhook.Allowed(admissionSpec, resourcePluralName, &status) {
		return status
	}

	admittingObject := &v1beta1.KubeFedCluster{}
	err := webhook.Unmarshal(&admissionSpec.Object, admittingObject, &status)
	if err != nil {
		return status
	}

	klog.V(4).Infof("Validating %q = %+v", ResourceName, *admittingObject)

	isStatusSubResource := admissionSpec.SubResource == "status"
	webhook.Validate(&status, func() field.ErrorList {
		return validation.ValidateKubeFedCluster(admittingObject, isStatusSubResource)
	})

	return status
}
