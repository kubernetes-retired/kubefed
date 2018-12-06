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
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/sets"
)

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
func GetOverrides(override *unstructured.Unstructured) (OverridesMap, error) {
	overridesMap := make(OverridesMap)
	if override == nil {
		return overridesMap, nil
	}

	overridesPath := fmt.Sprintf("%s.%s", SpecField, OverridesField)

	rawOverrides, ok, err := unstructured.NestedSlice(override.Object, SpecField, OverridesField)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve %s: %v", overridesPath, err)
	}
	if !ok {
		return nil, fmt.Errorf("%s not found", overridesPath)
	}

	for i, overrideInterface := range rawOverrides {
		rawOverride := overrideInterface.(map[string]interface{})

		rawClusterName, ok := rawOverride[ClusterNameField]
		if !ok {
			return nil, fmt.Errorf("%s not found for %s[%d]", ClusterNameField, overridesPath, i)
		}
		clusterName := rawClusterName.(string)
		if _, ok := overridesMap[clusterName]; ok {
			return nil, fmt.Errorf("cluster %q appears more than once in %s", clusterName, overridesPath)
		}
		overridesMap[clusterName] = make(ClusterOverridesMap)

		rawClusterOverrides, ok := rawOverride[ClusterOverridesField]
		if !ok {
			return nil, fmt.Errorf("%s not found for %s[%s]", ClusterOverridesField, overridesPath, clusterName)
		}
		clusterOverrides := rawClusterOverrides.([]interface{})

		for j, rawClusterOverride := range clusterOverrides {
			clusterOverride := rawClusterOverride.(map[string]interface{})

			rawPath, ok := clusterOverride[PathField]
			if !ok {
				return nil, fmt.Errorf("%s not found for %s[%s].%s[%d]", PathField, overridesPath, clusterName, ClusterOverridesField, j)
			}
			path := rawPath.(string)
			if invalidPaths.Has(path) {
				return nil, fmt.Errorf("%s %q is invalid for %s[%s].%s[%d]", PathField, path, overridesPath, clusterName, ClusterOverridesField, j)
			}
			if _, ok := overridesMap[clusterName][path]; ok {
				return nil, fmt.Errorf("%s %q appears more than once for %s[%s]", PathField, path, overridesPath, clusterName)
			}

			value, ok := clusterOverride[ValueField]
			if !ok {
				return nil, fmt.Errorf("%s for %q not found for %s[%s].%s[%d]", ValueField, path, overridesPath, clusterName, ClusterOverridesField, j)
			}

			overridesMap[clusterName][path] = value
		}
	}

	return overridesMap, nil
}

// SetOverrides sets the spec.overrides field of the unstructured
// object from the provided overrides map.
func SetOverrides(override *unstructured.Unstructured, overridesMap OverridesMap) error {
	rawSpec := override.Object[SpecField]
	if rawSpec == nil {
		rawSpec = map[string]interface{}{}
		override.Object[SpecField] = rawSpec
	}

	spec, ok := rawSpec.(map[string]interface{})
	if !ok {
		return fmt.Errorf("Unable to set overrides since %q is not an object: %T", SpecField, rawSpec)
	}
	spec[OverridesField] = overridesMap.ToUnstructuredSlice()
	return nil
}
