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
	"errors"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func GetClusterNames(placement *unstructured.Unstructured) []string {
	// TODO (font): NestedStringSlice returns false if the clusternames field
	// value is not found, which can happen when the clusternames field is
	// empty i.e. when a user does not want to propagate the resource anywhere.
	// Therefore, ignore the ok return value for now as we'll expect false
	// returned only in the event the clusternames field is empty, which is a
	// valid use-case. Ideally, we should not avoid a false return and expand
	// or re-write NestedStringSlice to check for the empty case as well as to
	// make sure the unstructured object in-fact has a proper "spec" and
	// "clusternames" field to avoid any accidental typos in the creation of a
	// propagation resource.
	clusterNames, _ := unstructured.NestedStringSlice(placement.Object, "spec", "clusternames")
	return clusterNames
}

func SetClusterNames(placement *unstructured.Unstructured, clusterNames []string) error {
	ok := unstructured.SetNestedStringSlice(placement.Object, clusterNames, "spec", "clusternames")
	if !ok {
		return errors.New("Unable to set the spec.clusternames field.")
	}
	return nil
}
