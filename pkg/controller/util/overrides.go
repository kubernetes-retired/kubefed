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

	"github.com/pkg/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/sets"
)

type ClusterOverride struct {
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

type GenericOverrideItem struct {
	ClusterName      string            `json:"clusterName"`
	ClusterOverrides []ClusterOverride `json:"clusterOverrides,omitempty"`
}

type GenericOverrideSpec struct {
	Overrides []GenericOverrideItem `json:"overrides,omitempty"`
}

type GenericOverride struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec *GenericOverrideSpec `json:"spec,omitempty"`
}

// Namespace and name may not be overridden since these fields are the
// primary mechanism of association between a federated resource in
// the host cluster and the target resources in the member clusters.
var invalidPaths = sets.NewString(
	"metadata.namespace",
	"metadata.name",
	"metadata.generateName",
)

// Mapping of qualified path (e.g. spec.replicas) to value
type ClusterOverridesMap map[string]interface{}

// Mapping of clusterName to overrides for the cluster
type OverridesMap map[string]ClusterOverridesMap

// ToUnstructuredSlice converts the map of overrides to a slice of
// interfaces that can be set in an unstructured object.
func (m OverridesMap) ToUnstructuredSlice() []interface{} {
	overrides := []interface{}{}
	for clusterName, clusterOverridesMap := range m {
		clusterOverrides := []map[string]interface{}{}
		for path, value := range clusterOverridesMap {
			clusterOverrides = append(clusterOverrides, map[string]interface{}{
				PathField:  path,
				ValueField: value,
			})
		}
		overridesItem := map[string]interface{}{
			ClusterNameField:      clusterName,
			ClusterOverridesField: clusterOverrides,
		}
		overrides = append(overrides, overridesItem)
	}
	return overrides
}

// GetOverrides returns a map of overrides populated from the given
// unstructured object.
func GetOverrides(rawObj *unstructured.Unstructured) (OverridesMap, error) {
	overridesMap := make(OverridesMap)

	if rawObj == nil {
		return overridesMap, nil
	}

	override := GenericOverride{}
	err := UnstructuredToInterface(rawObj, &override)
	if err != nil {
		return nil, err
	}

	if override.Spec == nil || override.Spec.Overrides == nil {
		// No overrides defined for the federated type
		return overridesMap, nil
	}

	for _, overrideItem := range override.Spec.Overrides {
		clusterName := overrideItem.ClusterName
		if _, ok := overridesMap[clusterName]; ok {
			return nil, errors.Errorf("cluster %q appears more than once", clusterName)
		}
		overridesMap[clusterName] = make(ClusterOverridesMap)

		clusterOverrides := overrideItem.ClusterOverrides

		for i, clusterOverride := range clusterOverrides {
			path := clusterOverride.Path
			if invalidPaths.Has(path) {
				return nil, errors.Errorf("override[%d] for cluster %q has an invalid path: %s", i, clusterName, path)
			}
			if _, ok := overridesMap[clusterName][path]; ok {
				return nil, errors.Errorf("path %q appears more than once for cluster %q", path, clusterName)
			}

			overridesMap[clusterName][path] = clusterOverride.Value
		}
	}

	return overridesMap, nil
}

// SetOverrides sets the spec.overrides field of the unstructured
// object from the provided overrides map.
func SetOverrides(fedObject *unstructured.Unstructured, overridesMap OverridesMap) error {
	rawSpec := fedObject.Object[SpecField]
	if rawSpec == nil {
		rawSpec = map[string]interface{}{}
		fedObject.Object[SpecField] = rawSpec
	}

	spec, ok := rawSpec.(map[string]interface{})
	if !ok {
		return errors.Errorf("Unable to set overrides since %q is not an object: %T", SpecField, rawSpec)
	}
	spec[OverridesField] = overridesMap.ToUnstructuredSlice()
	return nil
}

// UnstructuredToInterface converts an unstructured object to the
// provided interface by json marshalling/unmarshalling.
func UnstructuredToInterface(rawObj *unstructured.Unstructured, obj interface{}) error {
	content, err := rawObj.MarshalJSON()
	if err != nil {
		return err
	}
	return json.Unmarshal(content, obj)
}
