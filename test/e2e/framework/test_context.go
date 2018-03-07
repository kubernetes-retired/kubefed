/*
Copyright 2017 The Kubernetes Authors.

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

package framework

import (
	"flag"
	"os"

	"github.com/golang/glog"
)

type TestContextType struct {
	TestManagedFederation bool
	InMemoryControllers   bool
	KubeConfig            string
	KubeContext           string
}

var TestContext TestContextType

func registerFlags(t *TestContextType) {
	flag.BoolVar(&t.TestManagedFederation, "test-managed-federation", false, "Whether the test run should rely on a test-managed federation.")
	flag.BoolVar(&t.InMemoryControllers, "in-memory-controllers", true, "Whether federation controllers should be started in memory.  Ignored if test-managed-federation is false.")
	flag.StringVar(&t.KubeConfig, "kubeconfig", os.Getenv("KUBECONFIG"), "Path to kubeconfig containing embedded authinfo.  Ignored if test-managed-federation is true.")
	flag.StringVar(&t.KubeContext, "context", "", "kubeconfig context to use/override. If unset, will use value from 'current-context'.")
}

func validateFlags(t *TestContextType) {
	if len(t.KubeConfig) == 0 {
		glog.Warning("kubeconfig not provided - enabling test-managed-federation.")
		t.TestManagedFederation = true
	} else if t.TestManagedFederation {
		glog.Warningf("kubeconfig %q will be ignored due to test-managed-federation being true.", t.KubeConfig)
	}

	if !t.TestManagedFederation && t.InMemoryControllers {
		glog.Info("in-memory-controllers require test-managed-federation=true - disabling.")
		t.InMemoryControllers = false
	}
}

func ParseFlags() {
	registerFlags(&TestContext)
	flag.Parse()
	validateFlags(&TestContext)
}
