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

package schedulingtypes

import (
	. "github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
)

type Scheduler interface {
	Start(stopChan <-chan struct{})
	HasSynced() bool
	Stop()
	Reconcile(obj pkgruntime.Object, qualifiedName QualifiedName) ReconciliationStatus
}
