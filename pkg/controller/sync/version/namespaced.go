/*
Copyright 2018 The Kubernetes Authors.

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

package version

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"

	fedv1a1 "sigs.k8s.io/kubefed/pkg/apis/core/v1alpha1"
	"sigs.k8s.io/kubefed/pkg/controller/util"
)

type namespacedVersionAdapter struct{}

func (*namespacedVersionAdapter) TypeName() string {
	return "PropagatedVersion"
}

func (*namespacedVersionAdapter) NewListObject() pkgruntime.Object {
	return &fedv1a1.PropagatedVersionList{}
}

func (*namespacedVersionAdapter) NewObject() pkgruntime.Object {
	return &fedv1a1.PropagatedVersion{}
}

func (*namespacedVersionAdapter) NewVersion(qualifiedName util.QualifiedName, ownerReference metav1.OwnerReference, status *fedv1a1.PropagatedVersionStatus) pkgruntime.Object {
	return &fedv1a1.PropagatedVersion{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       qualifiedName.Namespace,
			Name:            qualifiedName.Name,
			OwnerReferences: []metav1.OwnerReference{ownerReference},
		},
		Status: *status,
	}
}

func (*namespacedVersionAdapter) GetStatus(obj pkgruntime.Object) *fedv1a1.PropagatedVersionStatus {
	version := obj.(*fedv1a1.PropagatedVersion)
	status := version.Status
	return &status
}

func (*namespacedVersionAdapter) SetStatus(obj pkgruntime.Object, status *fedv1a1.PropagatedVersionStatus) {
	version := obj.(*fedv1a1.PropagatedVersion)
	version.Status = *status
}
