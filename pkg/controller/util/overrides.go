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

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
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

func GetOverridesMap(typeConfig typeconfig.Interface, override *unstructured.Unstructured) (OverridesMap, error) {
	overridesMap := make(OverridesMap)
	if override == nil || typeConfig.GetOverride() == nil {
		return overridesMap, nil
	}

	qualifiedName := NewQualifiedName(override)
	overrideKind := typeConfig.GetOverride().Kind

	rawOverrides, ok, err := unstructured.NestedSlice(override.Object, SpecField, OverridesField)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving spec.overrides for %s %q: %v", overrideKind, qualifiedName, err)
	}
	if !ok {
		return nil, fmt.Errorf("%s %q is missing spec.overrides: %v", overrideKind, qualifiedName, err)
	}

	for i, overrideInterface := range rawOverrides {
		rawOverride := overrideInterface.(map[string]interface{})

		rawClusterName, ok := rawOverride[ClusterNameField]
		if !ok {
			return nil, fmt.Errorf("%s %q is missing clusterName for spec.overrides[%d]", overrideKind, qualifiedName, i)
		}
		clusterName := rawClusterName.(string)
		if _, ok := overridesMap[clusterName]; ok {
			return nil, fmt.Errorf("Cluster %q appears more than once in %s %q", i, clusterName, overrideKind, qualifiedName)
		}
		overridesMap[clusterName] = make(ClusterOverridesMap)

		rawClusterOverrides, ok := rawOverride[ClusterOverridesField]
		if !ok {
			return nil, fmt.Errorf("%s %q is missing clusterOverrides for spec.overrides[%s]", overrideKind, qualifiedName, clusterName)
		}
		clusterOverrides := rawClusterOverrides.([]interface{})

		for j, rawClusterOverride := range clusterOverrides {
			clusterOverride := rawClusterOverride.(map[string]interface{})

			rawPath, ok := clusterOverride[PathField]
			if !ok {
				return nil, fmt.Errorf("%s %q is missing path for spec.overrides[%s].clusterOverrides[%d]", overrideKind, qualifiedName, clusterName, j)
			}
			path := rawPath.(string)
			if invalidPaths.Has(path) {
				return nil, fmt.Errorf("%s %q has an invalid path for spec.overrides[%s].clusterOverrides[%d]: %s", overrideKind, qualifiedName, clusterName, j, path)
			}
			if _, ok := overridesMap[clusterName][path]; ok {
				return nil, fmt.Errorf("Path %q appears more than once for cluster %q in %s %q", path, clusterName, overrideKind, qualifiedName)
			}

			value, ok := clusterOverride[ValueField]
			if !ok {
				return nil, fmt.Errorf("%s %q is missing the value for spec.overrides[%s].clusterOverrides[%s]", overrideKind, qualifiedName, clusterName, path)
			}

			overridesMap[clusterName][path] = value
		}
	}

	return overridesMap, nil
}
