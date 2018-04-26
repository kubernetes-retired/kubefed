/*
Copyright 2017 The Federation v2 Authors.

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
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/federation/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset_generated/clientset"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	kubeclientset "k8s.io/client-go/kubernetes"
)

const (
	SecretKind          = "Secret"
	FederatedSecretKind = "FederatedSecret"
)

func init() {
	RegisterFederatedTypeConfig(FederatedSecretKind, NewFederatedSecretAdapter)
	RegisterTestObjectsFunc(FederatedSecretKind, NewFederatedSecretObjectsForTest)
}

type FederatedSecretAdapter struct {
	client fedclientset.Interface
}

func NewFederatedSecretAdapter(client fedclientset.Interface) FederatedTypeAdapter {
	return &FederatedSecretAdapter{client: client}
}

func (a *FederatedSecretAdapter) FedClient() fedclientset.Interface {
	return a.client
}

func (a *FederatedSecretAdapter) Template() FedApiAdapter {
	return NewFederatedSecretTemplate(a.client)
}

func (a *FederatedSecretAdapter) Placement() PlacementAdapter {
	return NewFederatedSecretPlacement(a.client)
}

func (a *FederatedSecretAdapter) PlacementGroupVersionResource() schema.GroupVersionResource {
	return groupVersionResource("federatedsecretplacements")
}

func (a *FederatedSecretAdapter) Override() OverrideAdapter {
	return NewFederatedSecretOverride(a.client)
}

func (a *FederatedSecretAdapter) Target() TargetAdapter {
	return SecretAdapter{}
}

// TODO(marun) Copy the whole thing
func (a *FederatedSecretAdapter) ObjectForCluster(template, override pkgruntime.Object, clusterName string) pkgruntime.Object {
	fedSecret := template.(*fedv1a1.FederatedSecret)
	templateSecret := fedSecret.Spec.Template

	data := templateSecret.Data
	if override != nil {
		secretOverride := override.(*fedv1a1.FederatedSecretOverride)
		for _, clusterOverride := range secretOverride.Spec.Overrides {
			if clusterOverride.ClusterName == clusterName {
				data = clusterOverride.Data
				break
			}
		}
	}

	secret := &corev1.Secret{
		ObjectMeta: util.DeepCopyRelevantObjectMeta(templateSecret.ObjectMeta),
		Data:       data,
		Type:       templateSecret.Type,
	}

	// Avoid having to duplicate these details in the template or have
	// the name/namespace vary between the federation api and member
	// clusters.
	//
	// TODO(marun) this should be documented
	secret.Name = fedSecret.Name
	secret.Namespace = fedSecret.Namespace

	return secret
}

func (a *FederatedSecretAdapter) ObjectForUpdateOp(desiredObj, clusterObj pkgruntime.Object) pkgruntime.Object {
	return desiredObj
}

type FederatedSecretTemplate struct {
	client fedclientset.Interface
}

func NewFederatedSecretTemplate(client fedclientset.Interface) FedApiAdapter {
	return &FederatedSecretTemplate{client: client}
}

func (a *FederatedSecretTemplate) Kind() string {
	return FederatedSecretKind
}

func (a *FederatedSecretTemplate) ObjectType() pkgruntime.Object {
	return &fedv1a1.FederatedSecret{}
}

func (a *FederatedSecretTemplate) Create(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedSecret := obj.(*fedv1a1.FederatedSecret)
	return a.client.FederationV1alpha1().FederatedSecrets(fedSecret.Namespace).Create(fedSecret)
}

func (a *FederatedSecretTemplate) Delete(qualifiedName QualifiedName, options *metav1.DeleteOptions) error {
	return a.client.FederationV1alpha1().FederatedSecrets(qualifiedName.Namespace).Delete(qualifiedName.Name, options)
}

func (a *FederatedSecretTemplate) Get(qualifiedName QualifiedName) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedSecrets(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
}

func (a *FederatedSecretTemplate) List(namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedSecrets(namespace).List(options)
}

func (a *FederatedSecretTemplate) Update(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedSecret := obj.(*fedv1a1.FederatedSecret)
	return a.client.FederationV1alpha1().FederatedSecrets(fedSecret.Namespace).Update(fedSecret)
}

func (a *FederatedSecretTemplate) Watch(namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return a.client.FederationV1alpha1().FederatedSecrets(namespace).Watch(options)
}

type FederatedSecretPlacement struct {
	client fedclientset.Interface
}

func NewFederatedSecretPlacement(client fedclientset.Interface) PlacementAdapter {
	return &FederatedSecretPlacement{client: client}
}

func (a *FederatedSecretPlacement) Kind() string {
	return "FederatedSecretPlacement"
}

func (a *FederatedSecretPlacement) ObjectType() pkgruntime.Object {
	return &fedv1a1.FederatedSecretPlacement{}
}

func (a *FederatedSecretPlacement) Create(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedSecretPlacement := obj.(*fedv1a1.FederatedSecretPlacement)
	return a.client.FederationV1alpha1().FederatedSecretPlacements(fedSecretPlacement.Namespace).Create(fedSecretPlacement)
}

func (a *FederatedSecretPlacement) Delete(qualifiedName QualifiedName, options *metav1.DeleteOptions) error {
	return a.client.FederationV1alpha1().FederatedSecretPlacements(qualifiedName.Namespace).Delete(qualifiedName.Name, options)
}

func (a *FederatedSecretPlacement) Get(qualifiedName QualifiedName) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedSecretPlacements(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
}

func (a *FederatedSecretPlacement) List(namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedSecretPlacements(namespace).List(options)
}

func (a *FederatedSecretPlacement) Update(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedSecretPlacement := obj.(*fedv1a1.FederatedSecretPlacement)
	return a.client.FederationV1alpha1().FederatedSecretPlacements(fedSecretPlacement.Namespace).Update(fedSecretPlacement)
}

func (a *FederatedSecretPlacement) Watch(namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return a.client.FederationV1alpha1().FederatedSecretPlacements(namespace).Watch(options)
}

func (a *FederatedSecretPlacement) ClusterNames(obj pkgruntime.Object) []string {
	fedSecretPlacement := obj.(*fedv1a1.FederatedSecretPlacement)
	clusterNames := []string{}
	for _, name := range fedSecretPlacement.Spec.ClusterNames {
		clusterNames = append(clusterNames, name)
	}
	return clusterNames
}

func (a *FederatedSecretPlacement) SetClusterNames(obj pkgruntime.Object, clusterNames []string) {
	fedSecretPlacement := obj.(*fedv1a1.FederatedSecretPlacement)
	fedSecretPlacement.Spec.ClusterNames = clusterNames
}

type FederatedSecretOverride struct {
	client fedclientset.Interface
}

func NewFederatedSecretOverride(client fedclientset.Interface) OverrideAdapter {
	return &FederatedSecretOverride{client: client}
}

func (a *FederatedSecretOverride) Kind() string {
	return "FederatedSecretOverride"
}

func (a *FederatedSecretOverride) ObjectType() pkgruntime.Object {
	return &fedv1a1.FederatedSecretOverride{}
}

func (a *FederatedSecretOverride) Create(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedSecretOverride := obj.(*fedv1a1.FederatedSecretOverride)
	return a.client.FederationV1alpha1().FederatedSecretOverrides(fedSecretOverride.Namespace).Create(fedSecretOverride)
}

func (a *FederatedSecretOverride) Delete(qualifiedName QualifiedName, options *metav1.DeleteOptions) error {
	return a.client.FederationV1alpha1().FederatedSecretOverrides(qualifiedName.Namespace).Delete(qualifiedName.Name, options)
}

func (a *FederatedSecretOverride) Get(qualifiedName QualifiedName) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedSecretOverrides(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
}

func (a *FederatedSecretOverride) List(namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedSecretOverrides(namespace).List(options)
}

func (a *FederatedSecretOverride) Update(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedSecretOverride := obj.(*fedv1a1.FederatedSecretOverride)
	return a.client.FederationV1alpha1().FederatedSecretOverrides(fedSecretOverride.Namespace).Update(fedSecretOverride)
}

func (a *FederatedSecretOverride) Watch(namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return a.client.FederationV1alpha1().FederatedSecretOverrides(namespace).Watch(options)
}

type SecretAdapter struct {
}

func (SecretAdapter) Kind() string {
	return SecretKind
}

func (SecretAdapter) ObjectType() pkgruntime.Object {
	return &corev1.Secret{}
}

func (SecretAdapter) VersionCompareType() util.VersionCompareType {
	return util.ResourceVersion
}

func (SecretAdapter) Create(client kubeclientset.Interface, obj pkgruntime.Object) (pkgruntime.Object, error) {
	secret := obj.(*corev1.Secret)
	return client.CoreV1().Secrets(secret.Namespace).Create(secret)
}

func (SecretAdapter) Delete(client kubeclientset.Interface, qualifiedName QualifiedName, options *metav1.DeleteOptions) error {
	return client.CoreV1().Secrets(qualifiedName.Namespace).Delete(qualifiedName.Name, options)
}

func (SecretAdapter) Get(client kubeclientset.Interface, qualifiedName QualifiedName) (pkgruntime.Object, error) {
	return client.CoreV1().Secrets(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
}

func (SecretAdapter) List(client kubeclientset.Interface, namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return client.CoreV1().Secrets(namespace).List(options)
}

func (SecretAdapter) Update(client kubeclientset.Interface, obj pkgruntime.Object) (pkgruntime.Object, error) {
	secret := obj.(*corev1.Secret)
	return client.CoreV1().Secrets(secret.Namespace).Update(secret)
}

func (SecretAdapter) Watch(client kubeclientset.Interface, namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return client.CoreV1().Secrets(namespace).Watch(options)
}

func NewFederatedSecretObjectsForTest(namespace string, clusterNames []string) (template, placement, override pkgruntime.Object) {
	template = &fedv1a1.FederatedSecret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-secret-",
			Namespace:    namespace,
		},
		Spec: fedv1a1.FederatedSecretSpec{
			Template: corev1.Secret{
				Data: map[string][]byte{
					"A": []byte("ala ma kota"),
				},
				Type: corev1.SecretTypeOpaque,
			},
		},
	}
	placement = &fedv1a1.FederatedSecretPlacement{
		ObjectMeta: metav1.ObjectMeta{
			// Name will be set to match the template by the crud tester
			Namespace: namespace,
		},
		Spec: fedv1a1.FederatedSecretPlacementSpec{
			ClusterNames: clusterNames,
		},
	}

	s := "bar"
	var newData []byte
	copy(newData, s[:])
	clusterName := clusterNames[0]
	override = &fedv1a1.FederatedSecretOverride{
		ObjectMeta: metav1.ObjectMeta{
			// Name will be set to match the template by the crud tester
			Namespace: namespace,
		},
		Spec: fedv1a1.FederatedSecretOverrideSpec{
			Overrides: []fedv1a1.FederatedSecretClusterOverride{
				{
					ClusterName: clusterName,
					Data: map[string][]byte{
						"foo": newData,
					},
				},
			},
		},
	}
	return template, placement, override
}
