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

	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	corev1alpha1 "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned/typed/core/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
)

type namespacedVersionAdapter struct {
	client corev1alpha1.CoreV1alpha1Interface
}

func newNamespacedVersionAdapter(client fedclientset.Interface) VersionAdapter {
	return &namespacedVersionAdapter{client.CoreV1alpha1()}
}

func (a *namespacedVersionAdapter) TypeName() string {
	return "PropagatedVersion"
}

func (a *namespacedVersionAdapter) List(namespace string) (pkgruntime.Object, error) {
	return a.client.PropagatedVersions(namespace).List(metav1.ListOptions{})
}

func (a *namespacedVersionAdapter) NewVersion(qualifiedName util.QualifiedName, ownerReference metav1.OwnerReference, status *fedv1a1.PropagatedVersionStatus) pkgruntime.Object {
	return &fedv1a1.PropagatedVersion{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       qualifiedName.Namespace,
			Name:            qualifiedName.Name,
			OwnerReferences: []metav1.OwnerReference{ownerReference},
		},
		Status: *status,
	}
}

func (a *namespacedVersionAdapter) GetStatus(obj pkgruntime.Object) *fedv1a1.PropagatedVersionStatus {
	version := obj.(*fedv1a1.PropagatedVersion)
	status := version.Status
	return &status
}

func (a *namespacedVersionAdapter) SetStatus(obj pkgruntime.Object, status *fedv1a1.PropagatedVersionStatus) {
	version := obj.(*fedv1a1.PropagatedVersion)
	version.Status = *status
}

func (a *namespacedVersionAdapter) Create(obj pkgruntime.Object) (pkgruntime.Object, error) {
	version := obj.(*fedv1a1.PropagatedVersion)
	return a.client.PropagatedVersions(version.Namespace).Create(version)
}

func (a *namespacedVersionAdapter) Get(qualifiedName util.QualifiedName) (pkgruntime.Object, error) {
	return a.client.PropagatedVersions(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
}

func (a *namespacedVersionAdapter) UpdateStatus(obj pkgruntime.Object) (pkgruntime.Object, error) {
	version := obj.(*fedv1a1.PropagatedVersion)
	return a.client.PropagatedVersions(version.Namespace).UpdateStatus(version)
}
