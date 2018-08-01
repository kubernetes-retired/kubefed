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
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

type FederatedReplicaSetAdapter struct {
	fedClient fedclientset.Interface
}

func NewFederatedReplicaSetAdapter(fedClient fedclientset.Interface) Adapter {
	return &FederatedReplicaSetAdapter{
		fedClient: fedClient,
	}
}

func (d *FederatedReplicaSetAdapter) TemplateObject() pkgruntime.Object {
	return &fedv1a1.FederatedReplicaSet{}
}

func (d *FederatedReplicaSetAdapter) TemplateList(namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return d.fedClient.CoreV1alpha1().FederatedReplicaSets(namespace).List(options)
}

func (d *FederatedReplicaSetAdapter) TemplateWatch(namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return d.fedClient.CoreV1alpha1().FederatedReplicaSets(namespace).Watch(options)
}

func (d *FederatedReplicaSetAdapter) OverrideObject() pkgruntime.Object {
	return &fedv1a1.FederatedReplicaSetOverride{}
}

func (d *FederatedReplicaSetAdapter) OverrideList(namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return d.fedClient.CoreV1alpha1().FederatedReplicaSetOverrides(namespace).List(options)
}

func (d *FederatedReplicaSetAdapter) OverrideWatch(namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return d.fedClient.CoreV1alpha1().FederatedReplicaSetOverrides(namespace).Watch(options)
}

func (d *FederatedReplicaSetAdapter) PlacementObject() pkgruntime.Object {
	return &fedv1a1.FederatedReplicaSetPlacement{}
}

func (d *FederatedReplicaSetAdapter) PlacementList(namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return d.fedClient.CoreV1alpha1().FederatedReplicaSetPlacements(namespace).List(options)
}

func (d *FederatedReplicaSetAdapter) PlacementWatch(namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return d.fedClient.CoreV1alpha1().FederatedReplicaSetPlacements(namespace).Watch(options)
}

// TODO: Below methods can also be made common among FederatedReplicaSet
// and FederatedDeployment using reflect if really needed.
func (d *FederatedReplicaSetAdapter) ReconcilePlacement(fedClient fedclientset.Interface, qualifiedName util.QualifiedName, newClusterNames []string) error {
	placement, err := fedClient.CoreV1alpha1().FederatedReplicaSetPlacements(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		newPlacement := &fedv1a1.FederatedReplicaSetPlacement{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: qualifiedName.Namespace,
				Name:      qualifiedName.Name,
			},
			Spec: fedv1a1.FederatedReplicaSetPlacementSpec{
				ClusterNames: newClusterNames,
			},
		}
		_, err := fedClient.CoreV1alpha1().FederatedReplicaSetPlacements(qualifiedName.Namespace).Create(newPlacement)
		return err
	}

	if PlacementUpdateNeeded(placement.Spec.ClusterNames, newClusterNames) {
		newPlacement := placement
		newPlacement.Spec.ClusterNames = newClusterNames
		_, err := fedClient.CoreV1alpha1().FederatedReplicaSetPlacements(qualifiedName.Namespace).Update(newPlacement)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *FederatedReplicaSetAdapter) ReconcileOverride(fedClient fedclientset.Interface, qualifiedName util.QualifiedName, result map[string]int64) error {
	override, err := fedClient.CoreV1alpha1().FederatedReplicaSetOverrides(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		newOverride := &fedv1a1.FederatedReplicaSetOverride{
			ObjectMeta: metav1.ObjectMeta{
				Name:      qualifiedName.Name,
				Namespace: qualifiedName.Namespace,
			},
			Spec: fedv1a1.FederatedReplicaSetOverrideSpec{},
		}

		for clusterName, replicas := range result {
			var r int32 = int32(replicas)
			clusterOverride := fedv1a1.FederatedReplicaSetClusterOverride{
				ClusterName: clusterName,
				Replicas:    &r,
			}
			newOverride.Spec.Overrides = append(newOverride.Spec.Overrides, clusterOverride)
		}

		_, err := fedClient.CoreV1alpha1().FederatedReplicaSetOverrides(qualifiedName.Namespace).Create(newOverride)
		return err
	}

	if ReplicaSetOverrideUpdateNeeded(override.Spec, result) {
		newOverride := override
		newOverride.Spec = fedv1a1.FederatedReplicaSetOverrideSpec{}
		for clusterName, replicas := range result {
			var r int32 = int32(replicas)
			clusterOverride := fedv1a1.FederatedReplicaSetClusterOverride{
				ClusterName: clusterName,
				Replicas:    &r,
			}
			newOverride.Spec.Overrides = append(newOverride.Spec.Overrides, clusterOverride)
		}
		_, err := fedClient.CoreV1alpha1().FederatedReplicaSetOverrides(qualifiedName.Namespace).Update(newOverride)
		if err != nil {
			return err
		}
	}

	return nil
}

// This assumes that there would be no duplicate clusternames
func ReplicaSetOverrideUpdateNeeded(overrideSpec fedv1a1.FederatedReplicaSetOverrideSpec, result map[string]int64) bool {
	resultLen := len(result)
	checkLen := 0
	for _, override := range overrideSpec.Overrides {
		replicas, ok := result[override.ClusterName]
		if !ok || (override.Replicas == nil) || (int32(replicas) != *override.Replicas) {
			return true
		}
		checkLen += 1
	}

	return checkLen != resultLen
}
