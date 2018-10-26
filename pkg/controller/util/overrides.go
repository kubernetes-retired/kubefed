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

type ClusterOverrides struct {
	Path      []string
	Overrides map[string]interface{}
}

func NewClusterOverrides(typeConfig typeconfig.Interface, override *unstructured.Unstructured) (*ClusterOverrides, error) {
	overrides, path, err := unmarshallOverrides(typeConfig, override)
	if err != nil {
		return nil, err
	}
	return &ClusterOverrides{
		Path:      path,
		Overrides: overrides,
	}, nil
}

func unmarshallOverrides(typeConfig typeconfig.Interface, override *unstructured.Unstructured) (map[string]interface{}, []string, error) {
	overrideMap := make(map[string]interface{})
	overridePath := []string{}
	if override == nil || typeConfig.GetOverride() == nil {
		return overrideMap, overridePath, nil
	}

	qualifiedName := NewQualifiedName(override)
	overrideKind := typeConfig.GetOverride().Kind

	rawOverrides, ok, err := unstructured.NestedSlice(override.Object, "spec", "overrides")
	if err != nil {
		return nil, nil, fmt.Errorf("Error retrieving spec.overrides for %s %q: %v", overrideKind, qualifiedName, err)
	}
	if !ok {
		return nil, nil, fmt.Errorf("Missing spec.overrides for %s %q: %v", overrideKind, qualifiedName, err)
	}

	overridePath = typeConfig.GetOverridePath()
	if len(overridePath) == 0 {
		return nil, nil, fmt.Errorf("Override path is missing for %q", typeConfig.GetTarget().Kind)
	}

	overrideField := overridePath[len(overridePath)-1]
	for _, overrideInterface := range rawOverrides {
		clusterOverride := overrideInterface.(map[string]interface{})
		rawName, ok := clusterOverride[ClusterNameField]
		if !ok {
			return nil, nil, fmt.Errorf("Missing cluster name field for %s %q", overrideKind, qualifiedName)
		}
		name := rawName.(string)
		data, ok := clusterOverride[overrideField]
		if !ok {
			return nil, nil, fmt.Errorf("Missing overrides field %q for %s %q", overrideField, overrideKind, qualifiedName)
		}
		overrideMap[name] = data
	}

	return overrideMap, overridePath, nil
}
