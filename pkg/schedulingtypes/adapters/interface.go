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

package adapters

import (
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	. "github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

type Adapter interface {
	TemplateObject() pkgruntime.Object
	TemplateList(namespace string, options metav1.ListOptions) (pkgruntime.Object, error)
	TemplateWatch(namespace string, options metav1.ListOptions) (watch.Interface, error)

	OverrideObject() pkgruntime.Object
	OverrideList(namespace string, options metav1.ListOptions) (pkgruntime.Object, error)
	OverrideWatch(namespace string, options metav1.ListOptions) (watch.Interface, error)

	PlacementObject() pkgruntime.Object
	PlacementList(namespace string, options metav1.ListOptions) (pkgruntime.Object, error)
	PlacementWatch(namespace string, options metav1.ListOptions) (watch.Interface, error)

	ReconcilePlacement(fedClient fedclientset.Interface, qualifiedName QualifiedName, newClusterNames []string) error
	ReconcileOverride(fedClient fedclientset.Interface, qualifiedName QualifiedName, result map[string]int64) error
}
