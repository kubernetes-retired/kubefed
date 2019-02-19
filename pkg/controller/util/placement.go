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

package util

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
)

type GenericPlacementFields struct {
	ClusterNames    []string              `json:"clusterNames,omitempty"`
	ClusterSelector *metav1.LabelSelector `json:"clusterSelector,omitempty"`
}

// TODO(marun) Consider removing this intermediate field.  It is only
// used for grouping.
type GenericPlacementSpec struct {
	Placement GenericPlacementFields `json:"placement,omitempty"`
}

type GenericPlacement struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec GenericPlacementSpec `json:"spec,omitempty"`
}

type PlacementDirective struct {
	ClusterNames    []string
	ClusterSelector labels.Selector
}

func GetPlacementDirective(resource *unstructured.Unstructured) (*PlacementDirective, error) {
	content, err := resource.MarshalJSON()
	if err != nil {
		return nil, err
	}
	placement := GenericPlacement{}
	err = json.Unmarshal(content, &placement)
	if err != nil {
		return nil, err
	}
	selector, err := metav1.LabelSelectorAsSelector(placement.Spec.Placement.ClusterSelector)
	if err != nil {
		return nil, err
	}
	return &PlacementDirective{
		ClusterNames:    placement.Spec.Placement.ClusterNames,
		ClusterSelector: selector,
	}, nil
}

func GetClusterNames(fedObject *unstructured.Unstructured) ([]string, error) {
	clusterNames, _, err := unstructured.NestedStringSlice(fedObject.Object, SpecField, PlacementField, ClusterNamesField)
	return clusterNames, err
}

func SetClusterNames(fedObject *unstructured.Unstructured, clusterNames []string) error {
	return unstructured.SetNestedStringSlice(fedObject.Object, clusterNames, SpecField, PlacementField, ClusterNamesField)
}
