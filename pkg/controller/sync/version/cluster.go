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
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"

	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	genericclient "github.com/kubernetes-sigs/federation-v2/pkg/client/generic"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
)

type clusterVersionAdapter struct {
	client genericclient.Client
}

func newClusterVersionAdapter(client genericclient.Client) VersionAdapter {
	return &clusterVersionAdapter{client}
}

func (a *clusterVersionAdapter) TypeName() string {
	return "ClusterPropagatedVersion"
}

func (a *clusterVersionAdapter) List(namespace string) (pkgruntime.Object, error) {
	clusterPropagatedVersionList := &fedv1a1.ClusterPropagatedVersionList{}
	err := a.client.List(context.TODO(), clusterPropagatedVersionList, namespace)
	return clusterPropagatedVersionList, err
}

func (a *clusterVersionAdapter) NewVersion(qualifiedName util.QualifiedName, ownerReference metav1.OwnerReference, status *fedv1a1.PropagatedVersionStatus) pkgruntime.Object {
	return &fedv1a1.ClusterPropagatedVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:            qualifiedName.Name,
			OwnerReferences: []metav1.OwnerReference{ownerReference},
		},
		Status: *status,
	}
}

func (a *clusterVersionAdapter) GetStatus(obj pkgruntime.Object) *fedv1a1.PropagatedVersionStatus {
	version := obj.(*fedv1a1.ClusterPropagatedVersion)
	status := version.Status
	return &status
}

func (a *clusterVersionAdapter) SetStatus(obj pkgruntime.Object, status *fedv1a1.PropagatedVersionStatus) {
	version := obj.(*fedv1a1.ClusterPropagatedVersion)
	version.Status = *status
}

func (a *clusterVersionAdapter) Create(obj pkgruntime.Object) (pkgruntime.Object, error) {
	version := obj.(*fedv1a1.ClusterPropagatedVersion)
	err := a.client.Create(context.TODO(), version)
	return version, err
}

func (a *clusterVersionAdapter) Get(qualifiedName util.QualifiedName) (pkgruntime.Object, error) {
	clusterPropagatedVersion := &fedv1a1.ClusterPropagatedVersion{}
	err := a.client.Get(context.TODO(), clusterPropagatedVersion, qualifiedName.Namespace, qualifiedName.Name)
	return clusterPropagatedVersion, err
}

func (a *clusterVersionAdapter) UpdateStatus(obj pkgruntime.Object) (pkgruntime.Object, error) {
	version := obj.(*fedv1a1.ClusterPropagatedVersion)
	err := a.client.UpdateStatus(context.TODO(), version)
	return version, err
}
