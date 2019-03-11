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
	"context"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/leaderelection"

	app "github.com/kubernetes-sigs/federation-v2/cmd/controller-manager/app/leaderelection"
	"github.com/kubernetes-sigs/federation-v2/cmd/controller-manager/app/options"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	"github.com/kubernetes-sigs/federation-v2/test/e2e/framework"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("Leader Elector", func() {
	f := framework.NewFederationFramework("leaderelection")
	tl := framework.NewE2ELogger()

	var leaderElection *util.LeaderElectionConfiguration
	var opts *options.Options

	BeforeEach(func() {
		leaderElection = &util.LeaderElectionConfiguration{
			LeaderElect:   true,
			LeaseDuration: time.Second,
			RenewDeadline: 500 * time.Millisecond,
			RetryPeriod:   100 * time.Millisecond,
			ResourceLock:  "configmaps",
		}

		opts = options.NewOptions()
		opts.Config = f.ControllerConfig()
		opts.LeaderElection = leaderElection
	})

	It("should chose secondary instance, primary goes down", func() {
		if !framework.TestContext.TestManagedFederation {
			framework.Skipf("leader election is valid only in managed federation setup")
		}

		primaryLeaderElector, err := app.NewFederationLeaderElector(opts, func(opts *options.Options, stopChan <-chan struct{}) {
			tl.Log("Run controllers of primary controller manager")
			<-stopChan
		})
		framework.ExpectNoError(err)

		ctx := context.Background()
		primaryContext, primaryContextCancel := context.WithCancel(ctx)

		tl.Log("Running primary instance of controller manager")
		go func() {
			primaryLeaderElector.Run(primaryContext)
			waitToBecomeLeader(tl, primaryLeaderElector)
			tl.Log("Primary instance is elected leader")
		}()

		secondaryLeaderElector, err := app.NewFederationLeaderElector(opts, func(opts *options.Options, stopChan <-chan struct{}) {
			tl.Log("Run controllers of secondary controller manager")
			<-stopChan
		})
		framework.ExpectNoError(err)

		SecondaryContext, secondaryContextCancel := context.WithCancel(ctx)
		tl.Log("Running secondary instance of controller manager")
		go func() {
			secondaryLeaderElector.Run(SecondaryContext)
			waitToBecomeLeader(tl, secondaryLeaderElector)
			tl.Log("Secondary instance is elected leader")
		}()

		// Stop primary instance of controller manager
		primaryContextCancel()

		secondaryContextCancel()
	})
})

func waitToBecomeLeader(tl common.TestLogger, lec *leaderelection.LeaderElector) {
	err := wait.PollImmediate(100*time.Millisecond, 10*time.Second, func() (bool, error) {
		if lec.IsLeader() {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		tl.Fatalf("Timed out waiting to become leader, err: %v", err)
	}
}
