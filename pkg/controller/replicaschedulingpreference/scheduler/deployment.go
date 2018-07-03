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

package scheduler

import (
	"reflect"
	"sort"

	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

type FederatedDeploymentAdapter struct {
	fedClient fedclientset.Interface
}

func NewFederatedDeploymentAdapter(fedClient fedclientset.Interface) SchedulerAdapter {
	return &FederatedDeploymentAdapter{
		fedClient: fedClient,
	}
}

func (d *FederatedDeploymentAdapter) TemplateObject() pkgruntime.Object {
	return &fedv1a1.FederatedDeployment{}
}

func (d *FederatedDeploymentAdapter) TemplateList(namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return d.fedClient.CoreV1alpha1().FederatedDeployments(namespace).List(options)
}

func (d *FederatedDeploymentAdapter) TemplateWatch(namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return d.fedClient.CoreV1alpha1().FederatedDeployments(namespace).Watch(options)
}

func (d *FederatedDeploymentAdapter) OverrideObject() pkgruntime.Object {
	return &fedv1a1.FederatedDeploymentOverride{}
}

func (d *FederatedDeploymentAdapter) OverrideList(namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return d.fedClient.CoreV1alpha1().FederatedDeploymentOverrides(namespace).List(options)
}

func (d *FederatedDeploymentAdapter) OverrideWatch(namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return d.fedClient.CoreV1alpha1().FederatedDeploymentOverrides(namespace).Watch(options)
}

func (d *FederatedDeploymentAdapter) PlacementObject() pkgruntime.Object {
	return &fedv1a1.FederatedDeploymentPlacement{}
}

func (d *FederatedDeploymentAdapter) PlacementList(namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return d.fedClient.CoreV1alpha1().FederatedDeploymentPlacements(namespace).List(options)
}

func (d *FederatedDeploymentAdapter) PlacementWatch(namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return d.fedClient.CoreV1alpha1().FederatedDeploymentPlacements(namespace).Watch(options)
}

// TODO: Below methods can also be made common among FederatedDeployment
// and FederatedReplicaset using reflect if really needed.
func (d *FederatedDeploymentAdapter) ReconcilePlacement(fedClient fedclientset.Interface, qualifiedName util.QualifiedName, newClusterNames []string) error {
	placement, err := fedClient.CoreV1alpha1().FederatedDeploymentPlacements(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		newPlacement := &fedv1a1.FederatedDeploymentPlacement{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: qualifiedName.Namespace,
				Name:      qualifiedName.Name,
			},
			Spec: fedv1a1.FederatedDeploymentPlacementSpec{
				ClusterNames: newClusterNames,
			},
		}
		_, err := fedClient.CoreV1alpha1().FederatedDeploymentPlacements(qualifiedName.Namespace).Create(newPlacement)
		return err
	}

	if PlacementUpdateNeeded(placement.Spec.ClusterNames, newClusterNames) {
		newPlacement := placement
		newPlacement.Spec.ClusterNames = newClusterNames
		_, err := fedClient.CoreV1alpha1().FederatedDeploymentPlacements(qualifiedName.Namespace).Update(newPlacement)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *FederatedDeploymentAdapter) ReconcileOverride(fedClient fedclientset.Interface, qualifiedName util.QualifiedName, result map[string]int64) error {
	override, err := fedClient.CoreV1alpha1().FederatedDeploymentOverrides(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		newOverride := &fedv1a1.FederatedDeploymentOverride{
			ObjectMeta: metav1.ObjectMeta{
				Name:      qualifiedName.Name,
				Namespace: qualifiedName.Namespace,
			},
			Spec: fedv1a1.FederatedDeploymentOverrideSpec{},
		}

		for clusterName, replicas := range result {
			var r int32 = int32(replicas)
			clusterOverride := fedv1a1.FederatedDeploymentClusterOverride{
				ClusterName: clusterName,
				Replicas:    &r,
			}
			newOverride.Spec.Overrides = append(newOverride.Spec.Overrides, clusterOverride)
		}

		_, err := fedClient.CoreV1alpha1().FederatedDeploymentOverrides(qualifiedName.Namespace).Create(newOverride)
		return err
	}

	if deploymentOverrideUpdateNeeded(override.Spec, result) {
		newOverride := override
		newOverride.Spec = fedv1a1.FederatedDeploymentOverrideSpec{}
		for clusterName, replicas := range result {
			var r int32 = int32(replicas)
			clusterOverride := fedv1a1.FederatedDeploymentClusterOverride{
				ClusterName: clusterName,
				Replicas:    &r,
			}
			newOverride.Spec.Overrides = append(newOverride.Spec.Overrides, clusterOverride)
		}
		_, err := fedClient.CoreV1alpha1().FederatedDeploymentOverrides(qualifiedName.Namespace).Update(newOverride)
		if err != nil {
			return err
		}
	}

	return nil
}

// These assume that there would be no duplicate clusternames
func PlacementUpdateNeeded(names, newNames []string) bool {
	sort.Strings(names)
	sort.Strings(newNames)
	return !reflect.DeepEqual(names, newNames)
}

func deploymentOverrideUpdateNeeded(overrideSpec fedv1a1.FederatedDeploymentOverrideSpec, result map[string]int64) bool {
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
