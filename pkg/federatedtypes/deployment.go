/*
Copyright 2018 The Federation v2 Authors.

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
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	kubeclientset "k8s.io/client-go/kubernetes"
)

const (
	DeploymentKind          = "Deployment"
	FederatedDeploymentKind = "FederatedDeployment"
)

var (
	deploymentNamespaced bool                = true
	DeploymentTypeConfig FederatedTypeConfig = FederatedTypeConfig{
		ComparisonType: util.Generation,
		Template: FederationAPIResource{
			APIResource: apiResource(FederatedDeploymentKind, "federateddeployments", deploymentNamespaced),
		},
		Placement: FederationAPIResource{
			APIResource: apiResource("FederatedDeploymentPlacement", "federateddeploymentplacements", deploymentNamespaced),
		},
		Override: &FederationAPIResource{
			APIResource: apiResource("FederatedDeploymentOverride", "federateddeploymentoverrides", deploymentNamespaced),
		},
		Target: metav1.APIResource{
			Name:       "deployments",
			Group:      "apps",
			Kind:       DeploymentKind,
			Version:    "v1",
			Namespaced: deploymentNamespaced,
		},
		AdapterFactory: NewFederatedDeploymentAdapter,
	}
)

func init() {
	RegisterFederatedTypeConfig(FederatedDeploymentKind, DeploymentTypeConfig)
}

type FederatedDeploymentAdapter struct {
	client fedclientset.Interface
}

func NewFederatedDeploymentAdapter(client fedclientset.Interface) FederatedTypeAdapter {
	return &FederatedDeploymentAdapter{client: client}
}

func (a *FederatedDeploymentAdapter) FedClient() fedclientset.Interface {
	return a.client
}

func (a *FederatedDeploymentAdapter) Template() FedApiAdapter {
	return NewFederatedDeploymentTemplate(a.client)
}

func (a *FederatedDeploymentAdapter) Placement() PlacementAdapter {
	return NewFederatedDeploymentPlacement(a.client)
}

func (a *FederatedDeploymentAdapter) PlacementAPIResource() *metav1.APIResource {
	return &DeploymentTypeConfig.Placement.APIResource
}

func (a *FederatedDeploymentAdapter) Override() OverrideAdapter {
	return NewFederatedDeploymentOverride(a.client)
}

func (a *FederatedDeploymentAdapter) Target() TargetAdapter {
	return DeploymentAdapter{}
}

// TODO(marun) Copy the whole thing
func (a *FederatedDeploymentAdapter) ObjectForCluster(template, override pkgruntime.Object, clusterName string) pkgruntime.Object {
	fedDeployment := template.(*fedv1a1.FederatedDeployment)
	templateDeployment := fedDeployment.Spec.Template

	deployment := &appsv1.Deployment{
		ObjectMeta: util.DeepCopyRelevantObjectMeta(templateDeployment.ObjectMeta),
		Spec:       *templateDeployment.Spec.DeepCopy(),
	}

	if override != nil {
		deploymentOverride := override.(*fedv1a1.FederatedDeploymentOverride)
		for _, clusterOverride := range deploymentOverride.Spec.Overrides {
			if clusterOverride.ClusterName == clusterName {
				deployment.Spec.Replicas = clusterOverride.Replicas
				break
			}
		}
	}

	// Avoid having to duplicate these details in the template or have
	// the name/namespace vary between the federation api and member
	// clusters.
	//
	// TODO(marun) Document this
	deployment.Name = fedDeployment.Name
	deployment.Namespace = fedDeployment.Namespace

	return deployment
}

func (a *FederatedDeploymentAdapter) ObjectForUpdateOp(desiredObj, clusterObj pkgruntime.Object) pkgruntime.Object {
	return desiredObj
}

type FederatedDeploymentTemplate struct {
	client fedclientset.Interface
}

func NewFederatedDeploymentTemplate(client fedclientset.Interface) FedApiAdapter {
	return &FederatedDeploymentTemplate{client: client}
}

func (a *FederatedDeploymentTemplate) Kind() string {
	return FederatedDeploymentKind
}

func (a *FederatedDeploymentTemplate) ObjectType() pkgruntime.Object {
	return &fedv1a1.FederatedDeployment{}
}

func (a *FederatedDeploymentTemplate) Create(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedDeployment := obj.(*fedv1a1.FederatedDeployment)
	return a.client.FederationV1alpha1().FederatedDeployments(fedDeployment.Namespace).Create(fedDeployment)
}

func (a *FederatedDeploymentTemplate) Delete(qualifiedName QualifiedName, options *metav1.DeleteOptions) error {
	return a.client.FederationV1alpha1().FederatedDeployments(qualifiedName.Namespace).Delete(qualifiedName.Name, options)
}

func (a *FederatedDeploymentTemplate) Get(qualifiedName QualifiedName) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedDeployments(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
}

func (a *FederatedDeploymentTemplate) List(namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedDeployments(namespace).List(options)
}

func (a *FederatedDeploymentTemplate) Update(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedDeployment := obj.(*fedv1a1.FederatedDeployment)
	updatedObj, err := a.client.FederationV1alpha1().FederatedDeployments(fedDeployment.Namespace).Update(fedDeployment)
	return updatedObj, err
}

func (a *FederatedDeploymentTemplate) Watch(namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return a.client.FederationV1alpha1().FederatedDeployments(namespace).Watch(options)
}

type FederatedDeploymentPlacement struct {
	client fedclientset.Interface
}

func NewFederatedDeploymentPlacement(client fedclientset.Interface) PlacementAdapter {
	return &FederatedDeploymentPlacement{client: client}
}

func (a *FederatedDeploymentPlacement) Kind() string {
	return "FederatedDeploymentPlacement"
}

func (a *FederatedDeploymentPlacement) ObjectType() pkgruntime.Object {
	return &fedv1a1.FederatedDeploymentPlacement{}
}

func (a *FederatedDeploymentPlacement) Create(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedDeploymentPlacement := obj.(*fedv1a1.FederatedDeploymentPlacement)
	return a.client.FederationV1alpha1().FederatedDeploymentPlacements(fedDeploymentPlacement.Namespace).Create(fedDeploymentPlacement)
}

func (a *FederatedDeploymentPlacement) Delete(qualifiedName QualifiedName, options *metav1.DeleteOptions) error {
	return a.client.FederationV1alpha1().FederatedDeploymentPlacements(qualifiedName.Namespace).Delete(qualifiedName.Name, options)
}

func (a *FederatedDeploymentPlacement) Get(qualifiedName QualifiedName) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedDeploymentPlacements(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
}

func (a *FederatedDeploymentPlacement) List(namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedDeploymentPlacements(namespace).List(options)
}

func (a *FederatedDeploymentPlacement) Update(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedDeploymentPlacement := obj.(*fedv1a1.FederatedDeploymentPlacement)
	return a.client.FederationV1alpha1().FederatedDeploymentPlacements(fedDeploymentPlacement.Namespace).Update(fedDeploymentPlacement)
}

func (a *FederatedDeploymentPlacement) Watch(namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return a.client.FederationV1alpha1().FederatedDeploymentPlacements(namespace).Watch(options)
}

func (a *FederatedDeploymentPlacement) ClusterNames(obj pkgruntime.Object) []string {
	fedDeploymentPlacement := obj.(*fedv1a1.FederatedDeploymentPlacement)
	clusterNames := []string{}
	for _, name := range fedDeploymentPlacement.Spec.ClusterNames {
		clusterNames = append(clusterNames, name)
	}
	return clusterNames
}

func (a *FederatedDeploymentPlacement) SetClusterNames(obj pkgruntime.Object, clusterNames []string) {
	fedDeploymentPlacement := obj.(*fedv1a1.FederatedDeploymentPlacement)
	fedDeploymentPlacement.Spec.ClusterNames = clusterNames
}

type FederatedDeploymentOverride struct {
	client fedclientset.Interface
}

func NewFederatedDeploymentOverride(client fedclientset.Interface) OverrideAdapter {
	return &FederatedDeploymentOverride{client: client}
}

func (a *FederatedDeploymentOverride) Kind() string {
	return "FederatedDeploymentOverride"
}

func (a *FederatedDeploymentOverride) ObjectType() pkgruntime.Object {
	return &fedv1a1.FederatedDeploymentOverride{}
}

func (a *FederatedDeploymentOverride) Create(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedDeploymentOverride := obj.(*fedv1a1.FederatedDeploymentOverride)
	return a.client.FederationV1alpha1().FederatedDeploymentOverrides(fedDeploymentOverride.Namespace).Create(fedDeploymentOverride)
}

func (a *FederatedDeploymentOverride) Delete(qualifiedName QualifiedName, options *metav1.DeleteOptions) error {
	return a.client.FederationV1alpha1().FederatedDeploymentOverrides(qualifiedName.Namespace).Delete(qualifiedName.Name, options)
}

func (a *FederatedDeploymentOverride) Get(qualifiedName QualifiedName) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedDeploymentOverrides(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
}

func (a *FederatedDeploymentOverride) List(namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedDeploymentOverrides(namespace).List(options)
}

func (a *FederatedDeploymentOverride) Update(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedDeploymentOverride := obj.(*fedv1a1.FederatedDeploymentOverride)
	return a.client.FederationV1alpha1().FederatedDeploymentOverrides(fedDeploymentOverride.Namespace).Update(fedDeploymentOverride)
}

func (a *FederatedDeploymentOverride) Watch(namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return a.client.FederationV1alpha1().FederatedDeploymentOverrides(namespace).Watch(options)
}

type DeploymentAdapter struct {
}

func (DeploymentAdapter) Kind() string {
	return DeploymentKind
}

func (DeploymentAdapter) ObjectType() pkgruntime.Object {
	return &appsv1.Deployment{}
}

func (DeploymentAdapter) VersionCompareType() util.VersionCompareType {
	return DeploymentTypeConfig.ComparisonType
}

func (DeploymentAdapter) Create(client kubeclientset.Interface, obj pkgruntime.Object) (pkgruntime.Object, error) {
	deployment := obj.(*appsv1.Deployment)
	createdObj, err := client.AppsV1().Deployments(deployment.Namespace).Create(deployment)
	return createdObj, err
}

func (DeploymentAdapter) Delete(client kubeclientset.Interface, qualifiedName QualifiedName, options *metav1.DeleteOptions) error {
	return client.AppsV1().Deployments(qualifiedName.Namespace).Delete(qualifiedName.Name, options)
}

func (DeploymentAdapter) Get(client kubeclientset.Interface, qualifiedName QualifiedName) (pkgruntime.Object, error) {
	return client.AppsV1().Deployments(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
}

func (DeploymentAdapter) List(client kubeclientset.Interface, namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return client.AppsV1().Deployments(namespace).List(options)
}

func (DeploymentAdapter) Update(client kubeclientset.Interface, obj pkgruntime.Object) (pkgruntime.Object, error) {
	deployment := obj.(*appsv1.Deployment)
	return client.AppsV1().Deployments(deployment.Namespace).Update(deployment)
}
func (DeploymentAdapter) Watch(client kubeclientset.Interface, namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return client.AppsV1().Deployments(namespace).Watch(options)
}
