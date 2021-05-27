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
	"sigs.k8s.io/controller-runtime/pkg/client"

	fedv1a1 "sigs.k8s.io/kubefed/pkg/apis/core/v1alpha1"
	"sigs.k8s.io/kubefed/pkg/controller/util"
)

type VersionAdapter interface {
	TypeName() string

	// Create an empty instance of the version type
	NewObject() client.Object
	// Create an empty instance of list version type
	NewListObject() client.ObjectList
	// Create a populated instance of the version type
	NewVersion(qualifiedName util.QualifiedName, ownerReference metav1.OwnerReference, status *fedv1a1.PropagatedVersionStatus) client.Object

	// Type-agnostic access / mutation of the Status field of a version resource
	GetStatus(obj pkgruntime.Object) *fedv1a1.PropagatedVersionStatus
	SetStatus(obj pkgruntime.Object, status *fedv1a1.PropagatedVersionStatus)
}

func NewVersionAdapter(namespaced bool) VersionAdapter {
	if namespaced {
		return &namespacedVersionAdapter{}
	}
	return &clusterVersionAdapter{}
}
