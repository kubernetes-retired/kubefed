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
	pkgruntime "k8s.io/apimachinery/pkg/runtime"

	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
)

type VersionAdapter interface {
	TypeName() string

	// Create a new instance of the version type
	NewVersion(qualifiedName util.QualifiedName, status *fedv1a1.PropagatedVersionStatus) pkgruntime.Object

	// Type-agnostic access / mutation of the Status field of a version resource
	GetStatus(obj pkgruntime.Object) *fedv1a1.PropagatedVersionStatus
	SetStatus(obj pkgruntime.Object, status *fedv1a1.PropagatedVersionStatus)

	// Methods that interact with the API
	Create(obj pkgruntime.Object) (pkgruntime.Object, error)
	Delete(qualifiedName util.QualifiedName) error
	Get(qualifiedName util.QualifiedName) (pkgruntime.Object, error)
	List(namespace string) (pkgruntime.Object, error)
	UpdateStatus(obj pkgruntime.Object) (pkgruntime.Object, error)
}
