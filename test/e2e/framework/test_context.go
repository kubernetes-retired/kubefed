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
	"fmt"
	"os"
	"time"

	"github.com/golang/glog"

	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
)

type TestContextType struct {
	TestManagedFederation           bool
	InMemoryControllers             bool
	KubeConfig                      string
	KubeContext                     string
	FederationSystemNamespace       string
	ClusterNamespace                string
	SingleCallTimeout               time.Duration
	LimitedScope                    bool
	LimitedScopeInMemoryControllers bool
	WaitForFinalization             bool
}

func (t *TestContextType) RunControllers() bool {
	return t.TestManagedFederation || t.InMemoryControllers
}

var TestContext *TestContextType = &TestContextType{}

func registerFlags(t *TestContextType) {
	flag.BoolVar(&t.TestManagedFederation, "test-managed-federation",
		false, "Whether the test run should rely on a test-managed federation.")
	flag.BoolVar(&t.InMemoryControllers, "in-memory-controllers", false,
		"Whether federation controllers should be started in memory. This works like a hybrid setup if test-managed-federation is false by launching the federation controllers outside the unmanaged cluster.")
	flag.StringVar(&t.KubeConfig, "kubeconfig", os.Getenv("KUBECONFIG"),
		"Path to kubeconfig containing embedded authinfo.  Ignored if test-managed-federation is true.")
	flag.StringVar(&t.KubeContext, "context", "",
		"kubeconfig context to use/override. If unset, will use value from 'current-context'.")
	flag.StringVar(&t.FederationSystemNamespace, "federation-namespace", util.DefaultFederationSystemNamespace,
		fmt.Sprintf("The namespace the federation control plane is deployed in.  If unset, will default to %q.", util.DefaultFederationSystemNamespace))
	flag.StringVar(&t.ClusterNamespace, "registry-namespace", util.MulticlusterPublicNamespace,
		fmt.Sprintf("The cluster registry namespace.  If unset, will default to %q.", util.MulticlusterPublicNamespace))
	flag.DurationVar(&t.SingleCallTimeout, "single-call-timeout", DefaultSingleCallTimeout,
		fmt.Sprintf("The maximum duration of a single call.  If unset, will default to %v", DefaultSingleCallTimeout))
	flag.BoolVar(&t.LimitedScope, "limited-scope", false, "Whether the federation namespace (configurable via --federation-namespace) will be the only target for federation.")
	flag.BoolVar(&t.LimitedScopeInMemoryControllers, "limited-scope-in-memory-controllers", true,
		"Whether federation controllers started in memory should target only the test namespace.  If debugging cluster-scoped federation outside of a test namespace, this should be set to false.")
	flag.BoolVar(&t.WaitForFinalization, "wait-for-finalization", true,
		"Whether the test suite should wait for finalization before stopping fixtures or exiting.  Setting this to false will speed up test execution but likely result in wedged namespaces and is only recommended for disposeable clusters.")
}

func validateFlags(t *TestContextType) {
	if len(t.KubeConfig) == 0 {
		glog.Warning("kubeconfig not provided - enabling test-managed federation.")
		t.TestManagedFederation = true
	} else if t.TestManagedFederation {
		glog.Warningf("kubeconfig %q will be ignored due to test-managed-federation being true.", t.KubeConfig)
	}

	if !t.TestManagedFederation && t.InMemoryControllers {
		glog.Info("in-memory-controllers=true while test-managed-federation=false - this will launch the federation controllers outside the unmanaged cluster.")
	}
}

func ParseFlags() {
	registerFlags(TestContext)
	flag.Parse()
	validateFlags(TestContext)
}
