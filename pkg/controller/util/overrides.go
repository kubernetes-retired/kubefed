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
)

type ClusterOverride struct {
	Path       []string
	FieldValue interface{}
}

type ClusterOverrides map[string][]ClusterOverride

func GetClusterOverrides(typeConfig typeconfig.Interface, override *unstructured.Unstructured) (ClusterOverrides, error) {
	overrideMap := make(map[string][]ClusterOverride)
	if override == nil || typeConfig.GetOverride() == nil {
		return overrideMap, nil
	}

	qualifiedName := NewQualifiedName(override)
	overrideKind := typeConfig.GetOverride().Kind

	rawOverrides, ok, err := unstructured.NestedSlice(override.Object, "spec", "overrides")
	if err != nil {
		return nil, fmt.Errorf("Error retrieving spec.overrides for %s %q: %v", overrideKind, qualifiedName, err)
	}
	if !ok {
		return nil, fmt.Errorf("Missing spec.overrides for %s %q: %v", overrideKind, qualifiedName, err)
	}

	overridePaths := typeConfig.GetOverridePaths()
	if len(overridePaths) == 0 {
		return nil, fmt.Errorf("Override paths are missing for %q", typeConfig.GetTarget().Kind)
	}

	for _, overrideInterface := range rawOverrides {
		clusterOverride := overrideInterface.(map[string]interface{})
		rawClusterName, ok := clusterOverride[ClusterNameField]
		if !ok {
			return nil, fmt.Errorf("Missing cluster name field for %s %q", overrideKind, qualifiedName)
		}
		clusterName := rawClusterName.(string)

		for overrideField, overridePath := range overridePaths {
			data, ok := clusterOverride[overrideField]
			if !ok {
				return nil, fmt.Errorf("Missing override field %q for %s %q", overrideField, overrideKind, qualifiedName)
			}
			overrideMap[clusterName] = append(overrideMap[clusterName],
				ClusterOverride{
					Path:       overridePath,
					FieldValue: data,
				})
		}
	}

	return overrideMap, nil
}
