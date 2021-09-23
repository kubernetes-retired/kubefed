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
	"context"
	"reflect"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeclientset "k8s.io/client-go/kubernetes"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/kubefed/pkg/controller/util"
)

// TestLogger defines operations common across different types of testing
type TestLogger interface {
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Log(args ...interface{})
	Logf(format string, args ...interface{})
}

func Equivalent(actual, desired runtimeclient.Object) bool {
	// Check for meta & spec equivalence
	if !util.ObjectMetaAndSpecEquivalent(actual, desired) {
		return false
	}

	// Check for status equivalence
	statusActual := reflect.ValueOf(actual).Elem().FieldByName("Status").Interface()
	statusDesired := reflect.ValueOf(desired).Elem().FieldByName("Status").Interface()
	return reflect.DeepEqual(statusActual, statusDesired)
}

// WaitForNamespace waits for namespace to be created in a cluster.
func WaitForNamespaceOrDie(tl TestLogger, client kubeclientset.Interface, clusterName, namespace string, interval, timeout time.Duration) {
	err := wait.PollImmediate(interval, timeout, func() (exist bool, err error) {
		_, err = client.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return false, nil
		}
		if err != nil {
			tl.Errorf("Error waiting for namespace %q to be created in cluster %q: %v",
				namespace, clusterName, err)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		tl.Fatalf("Timed out waiting for namespace %q to exist in cluster %q: %v",
			namespace, clusterName, err)
	}
}
