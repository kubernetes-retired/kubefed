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
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	kubeclientset "k8s.io/client-go/kubernetes"
)

const (
	ReplicaSetKind          = "ReplicaSet"
	FederatedReplicaSetKind = "FederatedReplicaSet"
)

func init() {
	RegisterFederatedTypeConfig(FederatedReplicaSetKind, NewFederatedReplicaSetAdapter)
	RegisterTestObjectsFunc(FederatedReplicaSetKind, NewFederatedReplicaSetObjectsForTest)
}

type FederatedReplicaSetAdapter struct {
	client fedclientset.Interface
}

func NewFederatedReplicaSetAdapter(client fedclientset.Interface) FederatedTypeAdapter {
	return &FederatedReplicaSetAdapter{client: client}
}

func (a *FederatedReplicaSetAdapter) FedClient() fedclientset.Interface {
	return a.client
}

func (a *FederatedReplicaSetAdapter) Template() FedApiAdapter {
	return NewFederatedReplicaSetTemplate(a.client)
}

func (a *FederatedReplicaSetAdapter) Placement() PlacementAdapter {
	return NewFederatedReplicaSetPlacement(a.client)
}

func (a *FederatedReplicaSetAdapter) PlacementAPIResource() *metav1.APIResource {
	return apiResource("federatedreplicasetplacements")
}

func (a *FederatedReplicaSetAdapter) Override() OverrideAdapter {
	return NewFederatedReplicaSetOverride(a.client)
}

func (a *FederatedReplicaSetAdapter) Target() TargetAdapter {
	return ReplicaSetAdapter{}
}

// TODO(marun) Copy the whole thing
func (a *FederatedReplicaSetAdapter) ObjectForCluster(template, override pkgruntime.Object, clusterName string) pkgruntime.Object {
	fedReplicaSet := template.(*fedv1a1.FederatedReplicaSet)
	templateReplicaSet := fedReplicaSet.Spec.Template

	replicaSet := &appsv1.ReplicaSet{
		ObjectMeta: util.DeepCopyRelevantObjectMeta(templateReplicaSet.ObjectMeta),
		Spec:       *templateReplicaSet.Spec.DeepCopy(),
	}

	if override != nil {
		replicaSetOverride := override.(*fedv1a1.FederatedReplicaSetOverride)
		for _, clusterOverride := range replicaSetOverride.Spec.Overrides {
			if clusterOverride.ClusterName == clusterName {
				replicaSet.Spec.Replicas = clusterOverride.Replicas
				break
			}
		}
	}

	// Avoid having to duplicate these details in the template or have
	// the name/namespace vary between the federation api and member
	// clusters.
	//
	// TODO(marun) Document this
	replicaSet.Name = fedReplicaSet.Name
	replicaSet.Namespace = fedReplicaSet.Namespace

	return replicaSet
}

func (a *FederatedReplicaSetAdapter) ObjectForUpdateOp(desiredObj, clusterObj pkgruntime.Object) pkgruntime.Object {
	return desiredObj
}

type FederatedReplicaSetTemplate struct {
	client fedclientset.Interface
}

func NewFederatedReplicaSetTemplate(client fedclientset.Interface) FedApiAdapter {
	return &FederatedReplicaSetTemplate{client: client}
}

func (a *FederatedReplicaSetTemplate) Kind() string {
	return FederatedReplicaSetKind
}

func (a *FederatedReplicaSetTemplate) ObjectType() pkgruntime.Object {
	return &fedv1a1.FederatedReplicaSet{}
}

func (a *FederatedReplicaSetTemplate) Create(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedReplicaSet := obj.(*fedv1a1.FederatedReplicaSet)
	return a.client.FederationV1alpha1().FederatedReplicaSets(fedReplicaSet.Namespace).Create(fedReplicaSet)
}

func (a *FederatedReplicaSetTemplate) Delete(qualifiedName QualifiedName, options *metav1.DeleteOptions) error {
	return a.client.FederationV1alpha1().FederatedReplicaSets(qualifiedName.Namespace).Delete(qualifiedName.Name, options)
}

func (a *FederatedReplicaSetTemplate) Get(qualifiedName QualifiedName) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedReplicaSets(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
}

func (a *FederatedReplicaSetTemplate) List(namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedReplicaSets(namespace).List(options)
}

func (a *FederatedReplicaSetTemplate) Update(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedReplicaSet := obj.(*fedv1a1.FederatedReplicaSet)
	updatedObj, err := a.client.FederationV1alpha1().FederatedReplicaSets(fedReplicaSet.Namespace).Update(fedReplicaSet)
	return updatedObj, err
}

func (a *FederatedReplicaSetTemplate) Watch(namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return a.client.FederationV1alpha1().FederatedReplicaSets(namespace).Watch(options)
}

type FederatedReplicaSetPlacement struct {
	client fedclientset.Interface
}

func NewFederatedReplicaSetPlacement(client fedclientset.Interface) PlacementAdapter {
	return &FederatedReplicaSetPlacement{client: client}
}

func (a *FederatedReplicaSetPlacement) Kind() string {
	return "FederatedReplicaSetPlacement"
}

func (a *FederatedReplicaSetPlacement) ObjectType() pkgruntime.Object {
	return &fedv1a1.FederatedReplicaSetPlacement{}
}

func (a *FederatedReplicaSetPlacement) Create(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedReplicaSetPlacement := obj.(*fedv1a1.FederatedReplicaSetPlacement)
	return a.client.FederationV1alpha1().FederatedReplicaSetPlacements(fedReplicaSetPlacement.Namespace).Create(fedReplicaSetPlacement)
}

func (a *FederatedReplicaSetPlacement) Delete(qualifiedName QualifiedName, options *metav1.DeleteOptions) error {
	return a.client.FederationV1alpha1().FederatedReplicaSetPlacements(qualifiedName.Namespace).Delete(qualifiedName.Name, options)
}

func (a *FederatedReplicaSetPlacement) Get(qualifiedName QualifiedName) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedReplicaSetPlacements(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
}

func (a *FederatedReplicaSetPlacement) List(namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedReplicaSetPlacements(namespace).List(options)
}

func (a *FederatedReplicaSetPlacement) Update(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedReplicaSetPlacement := obj.(*fedv1a1.FederatedReplicaSetPlacement)
	return a.client.FederationV1alpha1().FederatedReplicaSetPlacements(fedReplicaSetPlacement.Namespace).Update(fedReplicaSetPlacement)
}

func (a *FederatedReplicaSetPlacement) Watch(namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return a.client.FederationV1alpha1().FederatedReplicaSetPlacements(namespace).Watch(options)
}

func (a *FederatedReplicaSetPlacement) ClusterNames(obj pkgruntime.Object) []string {
	fedReplicaSetPlacement := obj.(*fedv1a1.FederatedReplicaSetPlacement)
	clusterNames := []string{}
	for _, name := range fedReplicaSetPlacement.Spec.ClusterNames {
		clusterNames = append(clusterNames, name)
	}
	return clusterNames
}

func (a *FederatedReplicaSetPlacement) SetClusterNames(obj pkgruntime.Object, clusterNames []string) {
	fedReplicaSetPlacement := obj.(*fedv1a1.FederatedReplicaSetPlacement)
	fedReplicaSetPlacement.Spec.ClusterNames = clusterNames
}

type FederatedReplicaSetOverride struct {
	client fedclientset.Interface
}

func NewFederatedReplicaSetOverride(client fedclientset.Interface) OverrideAdapter {
	return &FederatedReplicaSetOverride{client: client}
}

func (a *FederatedReplicaSetOverride) Kind() string {
	return "FederatedReplicaSetOverride"
}

func (a *FederatedReplicaSetOverride) ObjectType() pkgruntime.Object {
	return &fedv1a1.FederatedReplicaSetOverride{}
}

func (a *FederatedReplicaSetOverride) Create(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedReplicaSetOverride := obj.(*fedv1a1.FederatedReplicaSetOverride)
	return a.client.FederationV1alpha1().FederatedReplicaSetOverrides(fedReplicaSetOverride.Namespace).Create(fedReplicaSetOverride)
}

func (a *FederatedReplicaSetOverride) Delete(qualifiedName QualifiedName, options *metav1.DeleteOptions) error {
	return a.client.FederationV1alpha1().FederatedReplicaSetOverrides(qualifiedName.Namespace).Delete(qualifiedName.Name, options)
}

func (a *FederatedReplicaSetOverride) Get(qualifiedName QualifiedName) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedReplicaSetOverrides(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
}

func (a *FederatedReplicaSetOverride) List(namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedReplicaSetOverrides(namespace).List(options)
}

func (a *FederatedReplicaSetOverride) Update(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedReplicaSetOverride := obj.(*fedv1a1.FederatedReplicaSetOverride)
	return a.client.FederationV1alpha1().FederatedReplicaSetOverrides(fedReplicaSetOverride.Namespace).Update(fedReplicaSetOverride)
}

func (a *FederatedReplicaSetOverride) Watch(namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return a.client.FederationV1alpha1().FederatedReplicaSetOverrides(namespace).Watch(options)
}

type ReplicaSetAdapter struct {
}

func (ReplicaSetAdapter) Kind() string {
	return ReplicaSetKind
}

func (ReplicaSetAdapter) ObjectType() pkgruntime.Object {
	return &appsv1.ReplicaSet{}
}

func (ReplicaSetAdapter) VersionCompareType() util.VersionCompareType {
	return util.Generation
}

func (ReplicaSetAdapter) Create(client kubeclientset.Interface, obj pkgruntime.Object) (pkgruntime.Object, error) {
	replicaSet := obj.(*appsv1.ReplicaSet)
	createdObj, err := client.AppsV1().ReplicaSets(replicaSet.Namespace).Create(replicaSet)
	return createdObj, err
}

func (ReplicaSetAdapter) Delete(client kubeclientset.Interface, qualifiedName QualifiedName, options *metav1.DeleteOptions) error {
	return client.AppsV1().ReplicaSets(qualifiedName.Namespace).Delete(qualifiedName.Name, options)
}

func (ReplicaSetAdapter) Get(client kubeclientset.Interface, qualifiedName QualifiedName) (pkgruntime.Object, error) {
	return client.AppsV1().ReplicaSets(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
}

func (ReplicaSetAdapter) List(client kubeclientset.Interface, namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return client.AppsV1().ReplicaSets(namespace).List(options)
}

func (ReplicaSetAdapter) Update(client kubeclientset.Interface, obj pkgruntime.Object) (pkgruntime.Object, error) {
	replicaSet := obj.(*appsv1.ReplicaSet)
	return client.AppsV1().ReplicaSets(replicaSet.Namespace).Update(replicaSet)
}
func (ReplicaSetAdapter) Watch(client kubeclientset.Interface, namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return client.AppsV1().ReplicaSets(namespace).Watch(options)
}

func NewFederatedReplicaSetObjectsForTest(namespace string, clusterNames []string) (template, placement, override pkgruntime.Object) {
	replicas := int32(3)
	zero := int64(0)
	labels := map[string]string{"foo": "bar"}
	// TODO(marun) A replicaset created in a member cluster will have
	// some fields set to defaults if no value is provided for a given
	// field.  Unless a federated resource has all such fields
	// populated, a reconcile loop may result.  A loop would be
	// characterized by one or more fields being populated in the
	// member cluster resource but not in the federated resource,
	// resulting in endless attempts to update the member resource.
	// Possible workarounds include:
	//
	//   - performing the same defaulting in the fed api
	//   - avoid comparison of fields that are not populated
	//
	// As a temporary workaround, ensure all defaulted fields are
	// populated and mark them with comments.
	template = &fedv1a1.FederatedReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-replicaset-",
			Namespace:    namespace,
		},
		Spec: fedv1a1.FederatedReplicaSetSpec{
			Template: appsv1.ReplicaSet{
				Spec: appsv1.ReplicaSetSpec{
					Replicas: &replicas,
					Selector: &metav1.LabelSelector{
						MatchLabels: labels,
					},
					Template: apiv1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: labels,
						},
						Spec: apiv1.PodSpec{
							TerminationGracePeriodSeconds: &zero,
							Containers: []apiv1.Container{
								{
									Name:  "nginx",
									Image: "nginx",
								},
							},
						},
					},
				},
			},
		},
	}
	placement = &fedv1a1.FederatedReplicaSetPlacement{
		ObjectMeta: metav1.ObjectMeta{
			// Name will be set to match the template by the crud tester
			Namespace: namespace,
		},
		Spec: fedv1a1.FederatedReplicaSetPlacementSpec{
			ClusterNames: clusterNames,
		},
	}

	clusterName := clusterNames[0]
	clusterReplicas := int32(5)
	override = &fedv1a1.FederatedReplicaSetOverride{
		ObjectMeta: metav1.ObjectMeta{
			// Name will be set to match the template by the crud tester
			Namespace: namespace,
		},
		Spec: fedv1a1.FederatedReplicaSetOverrideSpec{
			Overrides: []fedv1a1.FederatedReplicaSetClusterOverride{
				{
					ClusterName: clusterName,
					Replicas:    &clusterReplicas,
				},
			},
		},
	}
	return template, placement, override
}
