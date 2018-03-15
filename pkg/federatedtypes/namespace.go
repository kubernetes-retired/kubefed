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

package federatedtypes

import (
	fedv1a1 "github.com/marun/fnord/pkg/apis/federation/v1alpha1"
	fedclientset "github.com/marun/fnord/pkg/client/clientset_generated/clientset"
	"github.com/marun/fnord/pkg/controller/util"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	kubeclientset "k8s.io/client-go/kubernetes"
)

const (
	NamespaceKind          = "Namespace"
	FederatedNamespaceKind = "FederatedNamespace"
)

func init() {
	RegisterFederatedTypeConfig(FederatedNamespaceKind, NewFederatedNamespaceAdapter)
	RegisterTestObjectsFunc(FederatedNamespaceKind, NewFederatedNamespaceObjectsForTest)
}

type FederatedNamespaceAdapter struct {
	client fedclientset.Interface
}

func NewFederatedNamespaceAdapter(client fedclientset.Interface) FederatedTypeAdapter {
	return &FederatedNamespaceAdapter{client: client}
}

func (a *FederatedNamespaceAdapter) Template() FedApiAdapter {
	return NewFederatedNamespaceTemplate(a.client)
}

func (a *FederatedNamespaceAdapter) Placement() PlacementAdapter {
	return NewFederatedNamespacePlacement(a.client)
}

func (a *FederatedNamespaceAdapter) Override() OverrideAdapter {
	return nil
}

func (a *FederatedNamespaceAdapter) Target() TargetAdapter {
	return NamespaceAdapter{}
}

// TODO(marun) Copy the whole thing
func (a *FederatedNamespaceAdapter) ObjectForCluster(template, override pkgruntime.Object, clusterName string) pkgruntime.Object {
	fedNamespace := template.(*fedv1a1.FederatedNamespace)
	templateNamespace := fedNamespace.Spec.Template

	namespace := &apiv1.Namespace{
		ObjectMeta: util.DeepCopyRelevantObjectMeta(templateNamespace.ObjectMeta),
		Spec:       *templateNamespace.Spec.DeepCopy(),
	}

	// Avoid having to duplicate these details in the template or have
	// the name vary between the federation api and member
	// clusters.
	//
	// TODO(marun) Document this
	namespace.Name = fedNamespace.Name

	return namespace
}

type FederatedNamespaceTemplate struct {
	client fedclientset.Interface
}

func NewFederatedNamespaceTemplate(client fedclientset.Interface) FedApiAdapter {
	return &FederatedNamespaceTemplate{client: client}
}

func (a *FederatedNamespaceTemplate) Kind() string {
	return FederatedNamespaceKind
}

func (a *FederatedNamespaceTemplate) ObjectMeta(obj pkgruntime.Object) *metav1.ObjectMeta {
	return &obj.(*fedv1a1.FederatedNamespace).ObjectMeta
}

func (a *FederatedNamespaceTemplate) ObjectType() pkgruntime.Object {
	return &fedv1a1.FederatedNamespace{}
}

func (a *FederatedNamespaceTemplate) Create(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedNamespace := obj.(*fedv1a1.FederatedNamespace)
	return a.client.FederationV1alpha1().FederatedNamespaces().Create(fedNamespace)
}

func (a *FederatedNamespaceTemplate) Delete(qualifiedName QualifiedName, options *metav1.DeleteOptions) error {
	return a.client.FederationV1alpha1().FederatedNamespaces().Delete(qualifiedName.Name, options)
}

func (a *FederatedNamespaceTemplate) Get(qualifiedName QualifiedName) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedNamespaces().Get(qualifiedName.Name, metav1.GetOptions{})
}

func (a *FederatedNamespaceTemplate) List(namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedNamespaces().List(options)
}

func (a *FederatedNamespaceTemplate) Update(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedNamespace := obj.(*fedv1a1.FederatedNamespace)
	return a.client.FederationV1alpha1().FederatedNamespaces().Update(fedNamespace)
}

func (a *FederatedNamespaceTemplate) Watch(namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return a.client.FederationV1alpha1().FederatedNamespaces().Watch(options)
}

type FederatedNamespacePlacement struct {
	client fedclientset.Interface
}

func NewFederatedNamespacePlacement(client fedclientset.Interface) PlacementAdapter {
	return &FederatedNamespacePlacement{client: client}
}

func (a *FederatedNamespacePlacement) Kind() string {
	return "FederatedNamespacePlacement"
}

func (a *FederatedNamespacePlacement) ObjectMeta(obj pkgruntime.Object) *metav1.ObjectMeta {
	return &obj.(*fedv1a1.FederatedNamespacePlacement).ObjectMeta
}

func (a *FederatedNamespacePlacement) ObjectType() pkgruntime.Object {
	return &fedv1a1.FederatedNamespacePlacement{}
}

func (a *FederatedNamespacePlacement) Create(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedNamespacePlacement := obj.(*fedv1a1.FederatedNamespacePlacement)
	return a.client.FederationV1alpha1().FederatedNamespacePlacements().Create(fedNamespacePlacement)
}

func (a *FederatedNamespacePlacement) Delete(qualifiedName QualifiedName, options *metav1.DeleteOptions) error {
	return a.client.FederationV1alpha1().FederatedNamespacePlacements().Delete(qualifiedName.Name, options)
}

func (a *FederatedNamespacePlacement) Get(qualifiedName QualifiedName) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedNamespacePlacements().Get(qualifiedName.Name, metav1.GetOptions{})
}

func (a *FederatedNamespacePlacement) List(namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedNamespacePlacements().List(options)
}

func (a *FederatedNamespacePlacement) Update(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedNamespacePlacement := obj.(*fedv1a1.FederatedNamespacePlacement)
	return a.client.FederationV1alpha1().FederatedNamespacePlacements().Update(fedNamespacePlacement)
}

func (a *FederatedNamespacePlacement) Watch(namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return a.client.FederationV1alpha1().FederatedNamespacePlacements().Watch(options)
}

func (a *FederatedNamespacePlacement) ClusterNames(obj pkgruntime.Object) []string {
	fedNamespacePlacement := obj.(*fedv1a1.FederatedNamespacePlacement)
	clusterNames := []string{}
	for _, name := range fedNamespacePlacement.Spec.ClusterNames {
		clusterNames = append(clusterNames, name)
	}
	return clusterNames
}

func (a *FederatedNamespacePlacement) SetClusterNames(obj pkgruntime.Object, clusterNames []string) {
	fedNamespacePlacement := obj.(*fedv1a1.FederatedNamespacePlacement)
	fedNamespacePlacement.Spec.ClusterNames = clusterNames
}

type NamespaceAdapter struct {
}

func (NamespaceAdapter) Kind() string {
	return NamespaceKind
}

func (NamespaceAdapter) ObjectMeta(obj pkgruntime.Object) *metav1.ObjectMeta {
	return &obj.(*apiv1.Namespace).ObjectMeta
}

func (NamespaceAdapter) ObjectType() pkgruntime.Object {
	return &apiv1.Namespace{}
}

func (NamespaceAdapter) Equivalent(obj1, obj2 pkgruntime.Object) bool {
	return util.ObjectMetaAndSpecEquivalent(obj1, obj2)
}

func (NamespaceAdapter) Create(client kubeclientset.Interface, obj pkgruntime.Object) (pkgruntime.Object, error) {
	namespace := obj.(*apiv1.Namespace)
	createdObj, err := client.CoreV1().Namespaces().Create(namespace)
	return createdObj, err
}

func (NamespaceAdapter) Delete(client kubeclientset.Interface, qualifiedName QualifiedName, options *metav1.DeleteOptions) error {
	return client.CoreV1().Namespaces().Delete(qualifiedName.Name, options)
}

func (NamespaceAdapter) Get(client kubeclientset.Interface, qualifiedName QualifiedName) (pkgruntime.Object, error) {
	return client.CoreV1().Namespaces().Get(qualifiedName.Name, metav1.GetOptions{})
}

func (NamespaceAdapter) List(client kubeclientset.Interface, namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return client.CoreV1().Namespaces().List(options)
}

func (NamespaceAdapter) Update(client kubeclientset.Interface, obj pkgruntime.Object) (pkgruntime.Object, error) {
	namespace := obj.(*apiv1.Namespace)
	return client.CoreV1().Namespaces().Update(namespace)
}
func (NamespaceAdapter) Watch(client kubeclientset.Interface, namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return client.CoreV1().Namespaces().Watch(options)
}

func NewFederatedNamespaceObjectsForTest(namespace string, clusterNames []string) (template, placement, override pkgruntime.Object) {
	template = &fedv1a1.FederatedNamespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-namespace-",
		},
		Spec: fedv1a1.FederatedNamespaceSpec{
			Template: apiv1.Namespace{
				Spec: apiv1.NamespaceSpec{
					Finalizers: []apiv1.FinalizerName{apiv1.FinalizerKubernetes},
				},
			},
		},
	}

	placement = &fedv1a1.FederatedNamespacePlacement{
		Spec: fedv1a1.FederatedNamespacePlacementSpec{
			ClusterNames: clusterNames,
		},
	}

	return template, placement, nil
}
