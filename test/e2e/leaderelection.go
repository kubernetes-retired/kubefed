/*
Copyright 2019 The Kubernetes Authors.

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

package e2e

import (
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"runtime"

	"sigs.k8s.io/kubefed/test/common"
	"sigs.k8s.io/kubefed/test/e2e/framework"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("Leader Elector", func() {
	f := framework.NewKubeFedFramework("leaderelection")
	tl := framework.NewE2ELogger()

	It("should chose secondary instance, primary goes down", func() {
		if framework.TestContext.LimitedScope {
			framework.Skipf("Testing of leader election requires an isolated test namespace which is only possible with a cluster-scoped control plane")
		}

		const leaderIdentifier = "promoted as leader"

		kubeConfigPath := framework.TestContext.KubeConfig
		systemNamespace := f.TestNamespaceName()

		primaryControllerManager, primaryLogStream, err := spawnControllerManagerProcess(tl, kubeConfigPath, systemNamespace)
		framework.ExpectNoError(err)
		if framework.WaitUntilLogStreamContains(tl, primaryLogStream, leaderIdentifier) {
			tl.Log("Primary controller manager became leader")
		} else {
			_ = primaryControllerManager.Process.Kill()
			tl.Fatal("Primary controller manager failed to become leader")
		}

		done := make(chan bool, 1)
		secondaryControllerManager, secondaryLogStream, err := spawnControllerManagerProcess(tl, kubeConfigPath, systemNamespace)
		framework.ExpectNoError(err)
		go func() {
			if framework.WaitUntilLogStreamContains(tl, secondaryLogStream, leaderIdentifier) {
				tl.Log("Secondary controller manager became leader")
				done <- true
			} else {
				_ = secondaryControllerManager.Process.Kill()
				tl.Fatal("Secondary controller manager failed to become leader")
			}
		}()

		err = primaryControllerManager.Process.Kill()
		framework.ExpectNoError(err)

		<-done

		err = secondaryControllerManager.Process.Kill()
		framework.ExpectNoError(err)
	})
})

func spawnControllerManagerProcess(tl common.TestLogger, kubeConfigPath, namespace string) (*exec.Cmd, io.ReadCloser, error) {
	// Get the directory of the current executable
	_, filename, _, _ := runtime.Caller(0)
	managedPath := filepath.Dir(filename)
	confFile, err := filepath.Abs(fmt.Sprintf("%s/../../config/kubefedconfig.yaml", managedPath))
	if err != nil {
		tl.Fatalf("Error discovering the configuration file for FecerationConfig resources: %v", err)
	}

	args := []string{fmt.Sprintf("--kubeconfig=%s", kubeConfigPath),
		fmt.Sprintf("--kubefed-namespace=%s", namespace),
		fmt.Sprintf("--kubefed-config=%s", confFile),
	}
	return framework.StartControllerManager(args)
}
