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

package common

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func NewTestObjects(typeConfig typeconfig.Interface, namespace string, clusterNames []string, fixture *unstructured.Unstructured) (template, placement, override *unstructured.Unstructured, err error) {
	template, err = NewTestTemplate(typeConfig.GetTemplate(), namespace, fixture)
	if err != nil {
		return nil, nil, nil, err
	}

	placement, err = newTestPlacement(typeConfig.GetPlacement(), namespace, clusterNames)
	if err != nil {
		return nil, nil, nil, err
	}

	overrideAPIResource := typeConfig.GetOverride()
	if overrideAPIResource != nil {
		override, err = newTestOverride(*overrideAPIResource, namespace, clusterNames, fixture)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	return template, placement, override, nil
}

func NewTestTemplate(apiResource metav1.APIResource, namespace string, fixture *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	obj := newTestUnstructured(apiResource, namespace)

	template, ok, err := unstructured.NestedFieldCopy(fixture.Object, "template")
	if err != nil {
		return nil, fmt.Errorf("Error retrieving template field: %v", err)
	}
	if ok {
		err := unstructured.SetNestedField(obj.Object, template, "spec", "template")
		if err != nil {
			return nil, err
		}
	}

	return obj, nil
}

func newTestOverride(apiResource metav1.APIResource, namespace string, clusterNames []string, fixture *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	obj := newTestUnstructured(apiResource, namespace)

	overridesSlice, ok, err := unstructured.NestedSlice(fixture.Object, "overrides")
	if err != nil {
		return nil, fmt.Errorf("Error retrieving overrides field: %v", err)
	}
	var targetOverrides map[string]interface{}
	if ok {
		targetOverrides = overridesSlice[0].(map[string]interface{})
	} else {
		targetOverrides = map[string]interface{}{}
	}
	targetOverrides[util.ClusterNameField] = clusterNames[0]
	overridesSlice[0] = targetOverrides
	err = unstructured.SetNestedSlice(obj.Object, overridesSlice, "spec", "overrides")
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func newTestPlacement(apiResource metav1.APIResource, namespace string, clusterNames []string) (*unstructured.Unstructured, error) {
	obj := newTestUnstructured(apiResource, namespace)

	err := util.SetClusterNames(obj, clusterNames)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func newTestUnstructured(apiResource metav1.APIResource, namespace string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetKind(apiResource.Kind)
	gv := schema.GroupVersion{Group: apiResource.Group, Version: apiResource.Version}
	obj.SetAPIVersion(gv.String())
	obj.SetGenerateName("test-e2e-")
	if apiResource.Namespaced {
		obj.SetNamespace(namespace)
	}
	return obj
}

func fixturePath() string {
	// Get the directory of the current executable
	_, filename, _, _ := runtime.Caller(0)
	commonPath := filepath.Dir(filename)
	return filepath.Join(commonPath, "fixtures")
}
