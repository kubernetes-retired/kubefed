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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	kubeclientset "k8s.io/client-go/kubernetes"

	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/federation/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset_generated/clientset"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
)

const (
	ServiceKind          = "Service"
	FederatedServiceKind = "FederatedService"
)

var (
	serviceNamespaced bool                = true
	ServiceTypeConfig FederatedTypeConfig = FederatedTypeConfig{
		ComparisonType: util.Generation,
		Template: FederationAPIResource{
			APIResource: apiResource(FederatedServiceKind, "federatedservices", serviceNamespaced),
		},
		Placement: FederationAPIResource{
			APIResource: apiResource("FederatedServicePlacement", "federatedserviceplacements", serviceNamespaced),
		},
		Target: metav1.APIResource{
			Name:       "services",
			Group:      "",
			Kind:       ServiceKind,
			Version:    "v1",
			Namespaced: serviceNamespaced,
		},
		AdapterFactory: NewFederatedServiceAdapter,
	}
)

func init() {
	RegisterFederatedTypeConfig(FederatedServiceKind, ServiceTypeConfig)
	RegisterTestObjectsFunc(FederatedServiceKind, NewFederatedServiceObjectsForTest)
}

type FederatedServiceAdapter struct {
	client fedclientset.Interface
}

func NewFederatedServiceAdapter(client fedclientset.Interface) FederatedTypeAdapter {
	return &FederatedServiceAdapter{client: client}
}

func (a *FederatedServiceAdapter) FedClient() fedclientset.Interface {
	return a.client
}

func (a *FederatedServiceAdapter) Template() FedApiAdapter {
	return NewFederatedServiceTemplate(a.client)
}

func (a *FederatedServiceAdapter) Placement() PlacementAdapter {
	return NewFederatedServicePlacement(a.client)
}

func (a *FederatedServiceAdapter) PlacementAPIResource() *metav1.APIResource {
	return &ServiceTypeConfig.Placement.APIResource
}

func (a *FederatedServiceAdapter) Override() OverrideAdapter {
	return nil
}

func (a *FederatedServiceAdapter) Target() TargetAdapter {
	return ServiceAdapter{}
}

// TODO(marun) copy the whole thing
func (a *FederatedServiceAdapter) ObjectForCluster(template, override pkgruntime.Object, clusterName string) pkgruntime.Object {
	fedService := template.(*fedv1a1.FederatedService)
	templateService := fedService.Spec.Template

	service := &corev1.Service{
		ObjectMeta: util.DeepCopyRelevantObjectMeta(templateService.ObjectMeta),
		Spec:       *templateService.Spec.DeepCopy(),
	}

	// Avoid having to duplicate these details in the template or have
	// the name/namespace vary between the federation api and member
	// clusters.
	//
	// TODO(marun) this should be documented
	service.Name = fedService.Name
	service.Namespace = fedService.Namespace

	return service
}

// TODO(shashi) Avoid the need for this adapter method by handling the scenario generically
// for cases where the field in spec is updated by controllers instead of status.
func (a *FederatedServiceAdapter) ObjectForUpdateOp(desiredObj, clusterObj pkgruntime.Object) pkgruntime.Object {
	desiredService := desiredObj.(*corev1.Service)
	clusterService := clusterObj.(*corev1.Service)

	// ClusterIP and NodePort are allocated to Service by cluster, so retain the same if any while updating
	desiredService.Spec.ClusterIP = clusterService.Spec.ClusterIP
	for i, fPort := range desiredService.Spec.Ports {
		for _, cPort := range clusterService.Spec.Ports {
			if fPort.Name == cPort.Name && fPort.Protocol == cPort.Protocol && fPort.Port == cPort.Port {
				desiredService.Spec.Ports[i].NodePort = cPort.NodePort
			}
		}
	}

	// Pass the same ResourceVersion as in the cluster object for update operation, otherwise operation will fail.
	desiredService.ResourceVersion = clusterService.ResourceVersion

	return desiredService
}

type FederatedServiceTemplate struct {
	client fedclientset.Interface
}

func NewFederatedServiceTemplate(client fedclientset.Interface) FedApiAdapter {
	return &FederatedServiceTemplate{client: client}
}

func (a *FederatedServiceTemplate) Kind() string {
	return FederatedServiceKind
}

func (a *FederatedServiceTemplate) ObjectType() pkgruntime.Object {
	return &fedv1a1.FederatedService{}
}

func (a *FederatedServiceTemplate) Create(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedService := obj.(*fedv1a1.FederatedService)
	return a.client.FederationV1alpha1().FederatedServices(fedService.Namespace).Create(fedService)
}

func (a *FederatedServiceTemplate) Delete(qualifiedName QualifiedName, options *metav1.DeleteOptions) error {
	return a.client.FederationV1alpha1().FederatedServices(qualifiedName.Namespace).Delete(qualifiedName.Name, options)
}

func (a *FederatedServiceTemplate) Get(qualifiedName QualifiedName) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedServices(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
}

func (a *FederatedServiceTemplate) List(namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedServices(namespace).List(options)
}

func (a *FederatedServiceTemplate) Update(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedService := obj.(*fedv1a1.FederatedService)
	return a.client.FederationV1alpha1().FederatedServices(fedService.Namespace).Update(fedService)
}

func (a *FederatedServiceTemplate) Watch(namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return a.client.FederationV1alpha1().FederatedServices(namespace).Watch(options)
}

type FederatedServicePlacement struct {
	client fedclientset.Interface
}

func NewFederatedServicePlacement(client fedclientset.Interface) PlacementAdapter {
	return &FederatedServicePlacement{client: client}
}

func (a *FederatedServicePlacement) Kind() string {
	return "FederatedServicePlacement"
}

func (a *FederatedServicePlacement) ObjectType() pkgruntime.Object {
	return &fedv1a1.FederatedServicePlacement{}
}

func (a *FederatedServicePlacement) Create(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedServicePlacement := obj.(*fedv1a1.FederatedServicePlacement)
	return a.client.FederationV1alpha1().FederatedServicePlacements(fedServicePlacement.Namespace).Create(fedServicePlacement)
}

func (a *FederatedServicePlacement) Delete(qualifiedName QualifiedName, options *metav1.DeleteOptions) error {
	return a.client.FederationV1alpha1().FederatedServicePlacements(qualifiedName.Namespace).Delete(qualifiedName.Name, options)
}

func (a *FederatedServicePlacement) Get(qualifiedName QualifiedName) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedServicePlacements(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
}

func (a *FederatedServicePlacement) List(namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedServicePlacements(namespace).List(options)
}

func (a *FederatedServicePlacement) Update(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedServicePlacement := obj.(*fedv1a1.FederatedServicePlacement)
	return a.client.FederationV1alpha1().FederatedServicePlacements(fedServicePlacement.Namespace).Update(fedServicePlacement)
}

func (a *FederatedServicePlacement) Watch(namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return a.client.FederationV1alpha1().FederatedServicePlacements(namespace).Watch(options)
}

func (a *FederatedServicePlacement) ClusterNames(obj pkgruntime.Object) []string {
	fedServicePlacement := obj.(*fedv1a1.FederatedServicePlacement)
	clusterNames := []string{}
	for _, name := range fedServicePlacement.Spec.ClusterNames {
		clusterNames = append(clusterNames, name)
	}
	return clusterNames
}

func (a *FederatedServicePlacement) SetClusterNames(obj pkgruntime.Object, clusterNames []string) {
	fedServicePlacement := obj.(*fedv1a1.FederatedServicePlacement)
	fedServicePlacement.Spec.ClusterNames = clusterNames
}

type ServiceAdapter struct {
}

func (ServiceAdapter) Kind() string {
	return ServiceKind
}

func (ServiceAdapter) ObjectType() pkgruntime.Object {
	return &corev1.Service{}
}

func (ServiceAdapter) VersionCompareType() util.VersionCompareType {
	return ServiceTypeConfig.ComparisonType
}

func (ServiceAdapter) Create(client kubeclientset.Interface, obj pkgruntime.Object) (pkgruntime.Object, error) {
	service := obj.(*corev1.Service)
	return client.CoreV1().Services(service.Namespace).Create(service)
}

func (ServiceAdapter) Delete(client kubeclientset.Interface, qualifiedName QualifiedName, options *metav1.DeleteOptions) error {
	return client.CoreV1().Services(qualifiedName.Namespace).Delete(qualifiedName.Name, options)
}

func (ServiceAdapter) Get(client kubeclientset.Interface, qualifiedName QualifiedName) (pkgruntime.Object, error) {
	return client.CoreV1().Services(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
}

func (ServiceAdapter) List(client kubeclientset.Interface, namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return client.CoreV1().Services(namespace).List(options)
}

func (ServiceAdapter) Update(client kubeclientset.Interface, obj pkgruntime.Object) (pkgruntime.Object, error) {
	service := obj.(*corev1.Service)
	return client.CoreV1().Services(service.Namespace).Update(service)
}

func (ServiceAdapter) Watch(client kubeclientset.Interface, namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return client.CoreV1().Services(namespace).Watch(options)
}

func NewFederatedServiceObjectsForTest(namespace string, clusterNames []string) (template, placement, override pkgruntime.Object) {
	template = &fedv1a1.FederatedService{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-service-",
			Namespace:    namespace,
		},
		Spec: fedv1a1.FederatedServiceSpec{
			Template: corev1.Service{
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeClusterIP,
					Ports: []corev1.ServicePort{
						{
							Name: "http",
							Port: 80,
						},
					},
				},
			},
		},
	}
	placement = &fedv1a1.FederatedServicePlacement{
		ObjectMeta: metav1.ObjectMeta{
			// Name will be set to match the template by the crud tester
			Namespace: namespace,
		},
		Spec: fedv1a1.FederatedServicePlacementSpec{
			ClusterNames: clusterNames,
		},
	}

	return template, placement, nil
}
