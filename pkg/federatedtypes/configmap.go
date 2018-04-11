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
	fedv1a1 "github.com/marun/federation-v2/pkg/apis/federation/v1alpha1"
	fedclientset "github.com/marun/federation-v2/pkg/client/clientset_generated/clientset"
	"github.com/marun/federation-v2/pkg/controller/util"
	apiv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	kubeclientset "k8s.io/client-go/kubernetes"
)

const (
	ConfigMapKind          = "ConfigMap"
	FederatedConfigMapKind = "FederatedConfigMap"
)

func init() {
	RegisterFederatedTypeConfig(FederatedConfigMapKind, NewFederatedConfigMapAdapter)
	RegisterTestObjectsFunc(FederatedConfigMapKind, NewFederatedConfigMapObjectsForTest)
}

type FederatedConfigMapAdapter struct {
	client fedclientset.Interface
}

func NewFederatedConfigMapAdapter(client fedclientset.Interface) FederatedTypeAdapter {
	return &FederatedConfigMapAdapter{client: client}
}

func (a *FederatedConfigMapAdapter) FedClient() fedclientset.Interface {
	return a.client
}

func (a *FederatedConfigMapAdapter) Template() FedApiAdapter {
	return NewFederatedConfigMapTemplate(a.client)
}

func (a *FederatedConfigMapAdapter) Placement() PlacementAdapter {
	return NewFederatedConfigMapPlacement(a.client)
}

func (a *FederatedConfigMapAdapter) PlacementGroupVersionResource() schema.GroupVersionResource {
	return groupVersionResource("federatedconfigmapplacements")
}

func (a *FederatedConfigMapAdapter) Override() OverrideAdapter {
	return NewFederatedConfigMapOverride(a.client)
}

func (a *FederatedConfigMapAdapter) Target() TargetAdapter {
	return ConfigMapAdapter{}
}

// TODO(marun) copy the whole thing
func (a *FederatedConfigMapAdapter) ObjectForCluster(template, override pkgruntime.Object, clusterName string) pkgruntime.Object {
	fedConfigMap := template.(*fedv1a1.FederatedConfigMap)
	templateConfigMap := fedConfigMap.Spec.Template

	data := templateConfigMap.Data
	if override != nil {
		configMapOverride := override.(*fedv1a1.FederatedConfigMapOverride)
		for _, clusterOverride := range configMapOverride.Spec.Overrides {
			if clusterOverride.ClusterName == clusterName {
				data = clusterOverride.Data
				break
			}
		}
	}

	configMap := &apiv1.ConfigMap{
		ObjectMeta: util.DeepCopyRelevantObjectMeta(templateConfigMap.ObjectMeta),
		Data:       data,
	}

	// Avoid having to duplicate these details in the template or have
	// the name/namespace vary between the federation api and member
	// clusters.
	//
	// TODO(marun) this should be documented
	configMap.Name = fedConfigMap.Name
	configMap.Namespace = fedConfigMap.Namespace

	return configMap
}

func (a *FederatedConfigMapAdapter) ObjectForUpdateOp(desiredObj, clusterObj pkgruntime.Object) pkgruntime.Object {
	return desiredObj
}

type FederatedConfigMapTemplate struct {
	client fedclientset.Interface
}

func NewFederatedConfigMapTemplate(client fedclientset.Interface) FedApiAdapter {
	return &FederatedConfigMapTemplate{client: client}
}

func (a *FederatedConfigMapTemplate) Kind() string {
	return FederatedConfigMapKind
}

func (a *FederatedConfigMapTemplate) ObjectMeta(obj pkgruntime.Object) *metav1.ObjectMeta {
	return &obj.(*fedv1a1.FederatedConfigMap).ObjectMeta
}

func (a *FederatedConfigMapTemplate) ObjectType() pkgruntime.Object {
	return &fedv1a1.FederatedConfigMap{}
}

func (a *FederatedConfigMapTemplate) Create(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedConfigMap := obj.(*fedv1a1.FederatedConfigMap)
	return a.client.FederationV1alpha1().FederatedConfigMaps(fedConfigMap.Namespace).Create(fedConfigMap)
}

func (a *FederatedConfigMapTemplate) Delete(qualifiedName QualifiedName, options *metav1.DeleteOptions) error {
	return a.client.FederationV1alpha1().FederatedConfigMaps(qualifiedName.Namespace).Delete(qualifiedName.Name, options)
}

func (a *FederatedConfigMapTemplate) Get(qualifiedName QualifiedName) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedConfigMaps(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
}

func (a *FederatedConfigMapTemplate) List(namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedConfigMaps(namespace).List(options)
}

func (a *FederatedConfigMapTemplate) Update(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedConfigMap := obj.(*fedv1a1.FederatedConfigMap)
	return a.client.FederationV1alpha1().FederatedConfigMaps(fedConfigMap.Namespace).Update(fedConfigMap)
}

func (a *FederatedConfigMapTemplate) Watch(namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return a.client.FederationV1alpha1().FederatedConfigMaps(namespace).Watch(options)
}

type FederatedConfigMapPlacement struct {
	client fedclientset.Interface
}

func NewFederatedConfigMapPlacement(client fedclientset.Interface) PlacementAdapter {
	return &FederatedConfigMapPlacement{client: client}
}

func (a *FederatedConfigMapPlacement) Kind() string {
	return "FederatedConfigMapPlacement"
}

func (a *FederatedConfigMapPlacement) ObjectMeta(obj pkgruntime.Object) *metav1.ObjectMeta {
	return &obj.(*fedv1a1.FederatedConfigMapPlacement).ObjectMeta
}

func (a *FederatedConfigMapPlacement) ObjectType() pkgruntime.Object {
	return &fedv1a1.FederatedConfigMapPlacement{}
}

func (a *FederatedConfigMapPlacement) Create(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedConfigMapPlacement := obj.(*fedv1a1.FederatedConfigMapPlacement)
	return a.client.FederationV1alpha1().FederatedConfigMapPlacements(fedConfigMapPlacement.Namespace).Create(fedConfigMapPlacement)
}

func (a *FederatedConfigMapPlacement) Delete(qualifiedName QualifiedName, options *metav1.DeleteOptions) error {
	return a.client.FederationV1alpha1().FederatedConfigMapPlacements(qualifiedName.Namespace).Delete(qualifiedName.Name, options)
}

func (a *FederatedConfigMapPlacement) Get(qualifiedName QualifiedName) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedConfigMapPlacements(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
}

func (a *FederatedConfigMapPlacement) List(namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedConfigMapPlacements(namespace).List(options)
}

func (a *FederatedConfigMapPlacement) Update(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedConfigMapPlacement := obj.(*fedv1a1.FederatedConfigMapPlacement)
	return a.client.FederationV1alpha1().FederatedConfigMapPlacements(fedConfigMapPlacement.Namespace).Update(fedConfigMapPlacement)
}

func (a *FederatedConfigMapPlacement) Watch(namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return a.client.FederationV1alpha1().FederatedConfigMapPlacements(namespace).Watch(options)
}

func (a *FederatedConfigMapPlacement) ClusterNames(obj pkgruntime.Object) []string {
	fedConfigMapPlacement := obj.(*fedv1a1.FederatedConfigMapPlacement)
	clusterNames := []string{}
	for _, name := range fedConfigMapPlacement.Spec.ClusterNames {
		clusterNames = append(clusterNames, name)
	}
	return clusterNames
}

func (a *FederatedConfigMapPlacement) SetClusterNames(obj pkgruntime.Object, clusterNames []string) {
	fedConfigMapPlacement := obj.(*fedv1a1.FederatedConfigMapPlacement)
	fedConfigMapPlacement.Spec.ClusterNames = clusterNames
}

type FederatedConfigMapOverride struct {
	client fedclientset.Interface
}

func NewFederatedConfigMapOverride(client fedclientset.Interface) OverrideAdapter {
	return &FederatedConfigMapOverride{client: client}
}

func (a *FederatedConfigMapOverride) Kind() string {
	return "FederatedConfigMapOverride"
}

func (a *FederatedConfigMapOverride) ObjectMeta(obj pkgruntime.Object) *metav1.ObjectMeta {
	return &obj.(*fedv1a1.FederatedConfigMapOverride).ObjectMeta
}

func (a *FederatedConfigMapOverride) ObjectType() pkgruntime.Object {
	return &fedv1a1.FederatedConfigMapOverride{}
}

func (a *FederatedConfigMapOverride) Create(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedConfigMapOverride := obj.(*fedv1a1.FederatedConfigMapOverride)
	return a.client.FederationV1alpha1().FederatedConfigMapOverrides(fedConfigMapOverride.Namespace).Create(fedConfigMapOverride)
}

func (a *FederatedConfigMapOverride) Delete(qualifiedName QualifiedName, options *metav1.DeleteOptions) error {
	return a.client.FederationV1alpha1().FederatedConfigMapOverrides(qualifiedName.Namespace).Delete(qualifiedName.Name, options)
}

func (a *FederatedConfigMapOverride) Get(qualifiedName QualifiedName) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedConfigMapOverrides(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
}

func (a *FederatedConfigMapOverride) List(namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedConfigMapOverrides(namespace).List(options)
}

func (a *FederatedConfigMapOverride) Update(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedConfigMapOverride := obj.(*fedv1a1.FederatedConfigMapOverride)
	return a.client.FederationV1alpha1().FederatedConfigMapOverrides(fedConfigMapOverride.Namespace).Update(fedConfigMapOverride)
}

func (a *FederatedConfigMapOverride) Watch(namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return a.client.FederationV1alpha1().FederatedConfigMapOverrides(namespace).Watch(options)
}

type ConfigMapAdapter struct {
}

func (ConfigMapAdapter) Kind() string {
	return ConfigMapKind
}

func (ConfigMapAdapter) ObjectMeta(obj pkgruntime.Object) *metav1.ObjectMeta {
	return &obj.(*apiv1.ConfigMap).ObjectMeta
}

func (ConfigMapAdapter) ObjectType() pkgruntime.Object {
	return &corev1.ConfigMap{}
}

func (ConfigMapAdapter) VersionCompareType() util.VersionCompareType {
	return util.ResourceVersion
}

func (ConfigMapAdapter) Create(client kubeclientset.Interface, obj pkgruntime.Object) (pkgruntime.Object, error) {
	configMap := obj.(*corev1.ConfigMap)
	return client.CoreV1().ConfigMaps(configMap.Namespace).Create(configMap)
}

func (ConfigMapAdapter) Delete(client kubeclientset.Interface, qualifiedName QualifiedName, options *metav1.DeleteOptions) error {
	return client.CoreV1().ConfigMaps(qualifiedName.Namespace).Delete(qualifiedName.Name, options)
}

func (ConfigMapAdapter) Get(client kubeclientset.Interface, qualifiedName QualifiedName) (pkgruntime.Object, error) {
	return client.CoreV1().ConfigMaps(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
}

func (ConfigMapAdapter) List(client kubeclientset.Interface, namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return client.CoreV1().ConfigMaps(namespace).List(options)
}

func (ConfigMapAdapter) Update(client kubeclientset.Interface, obj pkgruntime.Object) (pkgruntime.Object, error) {
	configMap := obj.(*corev1.ConfigMap)
	return client.CoreV1().ConfigMaps(configMap.Namespace).Update(configMap)
}

func (ConfigMapAdapter) Watch(client kubeclientset.Interface, namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return client.CoreV1().ConfigMaps(namespace).Watch(options)
}

func NewFederatedConfigMapObjectsForTest(namespace string, clusterNames []string) (template, placement, override pkgruntime.Object) {
	template = &fedv1a1.FederatedConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-config-map-",
			Namespace:    namespace,
		},
		Spec: fedv1a1.FederatedConfigMapSpec{
			Template: corev1.ConfigMap{
				Data: map[string]string{
					"A": "ala ma kota",
				},
			},
		},
	}
	placement = &fedv1a1.FederatedConfigMapPlacement{
		ObjectMeta: metav1.ObjectMeta{
			// Name will be set to match the template by the crud tester
			Namespace: namespace,
		},
		Spec: fedv1a1.FederatedConfigMapPlacementSpec{
			ClusterNames: clusterNames,
		},
	}
	clusterName := clusterNames[0]
	override = &fedv1a1.FederatedConfigMapOverride{
		ObjectMeta: metav1.ObjectMeta{
			// Name will be set to match the template by the crud tester
			Namespace: namespace,
		},
		Spec: fedv1a1.FederatedConfigMapOverrideSpec{
			Overrides: []fedv1a1.FederatedConfigMapClusterOverride{
				{
					ClusterName: clusterName,
					Data: map[string]string{
						"foo": "bar",
					},
				},
			},
		},
	}
	return template, placement, override
}
