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

package federatedtypes

import (
	"fmt"

	pkgruntime "k8s.io/apimachinery/pkg/runtime"
)

// NewTestObjectsFunc defines how to create the resources representing
// a federated type for testing purposes.
type NewTestObjectsFunc func(namespace string, clusterNames []string) (template, placement, override pkgruntime.Object)

var newTestObjFuncRegistry = make(map[string]NewTestObjectsFunc)

// RegisterTestObjectsFunc ensures that NewTestObjects() can create
// test objects for the given kind.
func RegisterTestObjectsFunc(kind string, objFunc NewTestObjectsFunc) {
	_, ok := newTestObjFuncRegistry[kind]
	if ok {
		// TODO Is panicking ok given that this is part of a type-registration mechanism
		panic(fmt.Sprintf("A new test object func for %q has already been registered", kind))
	}
	newTestObjFuncRegistry[kind] = objFunc
}

func NewTestObjects(kind, namespace string, clusterNames []string) (template, placement, override pkgruntime.Object) {
	f, ok := newTestObjFuncRegistry[kind]
	if !ok {
		panic(fmt.Sprintf("A test object func for %q has not been registered", kind))
	}
	return f(namespace, clusterNames)
}
