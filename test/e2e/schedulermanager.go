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
	"k8s.io/apimachinery/pkg/util/wait"
	restclient "k8s.io/client-go/rest"

	"sigs.k8s.io/kubefed/pkg/controller/schedulingmanager"
	"sigs.k8s.io/kubefed/pkg/controller/util"
	"sigs.k8s.io/kubefed/pkg/kubefedctl"
	kfenable "sigs.k8s.io/kubefed/pkg/kubefedctl/enable"
	"sigs.k8s.io/kubefed/pkg/schedulingtypes"
	"sigs.k8s.io/kubefed/test/common"
	"sigs.k8s.io/kubefed/test/e2e/framework"

	. "github.com/onsi/ginkgo" //nolint:stylecheck
)

var _ = Describe("Scheduler", func() {
	f := framework.NewKubeFedFramework("scheduler")
	tl := framework.NewE2ELogger()

	schedulingTypes := GetSchedulingTypes(tl)

	var namespace string
	var kubeConfig *restclient.Config
	var controllerFixture *framework.ControllerFixture
	var controller *schedulingmanager.SchedulingManager

	BeforeEach(func() {
		namespace = f.TestNamespaceName()

		if kubeConfig == nil {
			kubeConfig = f.KubeConfig()
		}
		if framework.TestContext.RunControllers() {
			config := f.ControllerConfig()
			config.KubeFedNamespace = namespace

			controllerFixture, controller = framework.NewSchedulingManagerFixture(tl, config)
			f.RegisterFixture(controllerFixture)
		}
	})

	Describe("Manager", func() {
		Context("when federatedtypeconfig resources are changed", func() {
			It("related scheduler and plugin controllers should be dynamically disabled/enabled", func() {
				if !framework.TestContext.RunControllers() {
					framework.Skipf("The scheduler manager can only be tested when controllers are running in-process.")
				}

				By("Enabling federatedtypeconfig resources again for scheduler/plugin controllers")
				for targetTypeName := range schedulingTypes {
					enableTypeConfigResource(targetTypeName, namespace, kubeConfig, tl)
				}
				enableTypeConfigResource(util.NamespaceName, namespace, kubeConfig, tl)

				// make sure scheduler/plugin initialization are done before our test
				By("Waiting for scheduler/plugin controllers are initialized in scheduling manager")
				waitForSchedulerStarted(tl, controller, schedulingTypes)

				By("Deleting federatedtypeconfig resources for scheduler/plugin controllers")
				for targetTypeName := range schedulingTypes {
					deleteTypeConfigResource(targetTypeName, namespace, kubeConfig, tl)
				}

				By("Waiting for scheduler/plugin controllers are destroyed in scheduling manager")
				waitForSchedulerDeleted(tl, controller)

				By("Enabling federatedtypeconfig resources again for scheduler/plugin controllers")
				for targetTypeName := range schedulingTypes {
					enableTypeConfigResource(targetTypeName, namespace, kubeConfig, tl)
				}

				By("Waiting for the scheduler/plugin controllers are started in scheduling manager")
				waitForSchedulerStarted(tl, controller, schedulingTypes)
			})
		})
	})

})

func GetSchedulingTypes(tl common.TestLogger) map[string]schedulingtypes.SchedulerFactory {
	schedulingTypes := make(map[string]schedulingtypes.SchedulerFactory)
	for typeConfigName, schedulingType := range schedulingtypes.SchedulingTypes() {
		if schedulingType.Kind != schedulingtypes.RSPKind {
			continue
		}
		schedulingTypes[typeConfigName] = schedulingType.SchedulerFactory
	}
	if len(schedulingTypes) == 0 {
		tl.Fatalf("No target types found for scheduling type %q", schedulingtypes.RSPKind)
	}

	return schedulingTypes
}

func waitForSchedulerDeleted(tl common.TestLogger, controller *schedulingmanager.SchedulingManager) {
	err := wait.PollImmediate(framework.PollInterval, framework.TestContext.SingleCallTimeout, func() (bool, error) {
		scheduler := controller.GetScheduler(schedulingtypes.RSPKind)
		if scheduler != nil {
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		tl.Fatalf("Error stopping for scheduler/plugin controllers: %v", err)
	}
}

func waitForSchedulerStarted(tl common.TestLogger, controller *schedulingmanager.SchedulingManager, schedulingTypes map[string]schedulingtypes.SchedulerFactory) {
	err := wait.PollImmediate(framework.PollInterval, framework.TestContext.SingleCallTimeout, func() (bool, error) {
		scheduler := controller.GetScheduler(schedulingtypes.RSPKind)
		if scheduler == nil {
			return false, nil
		}
		for targetTypeName := range schedulingTypes {
			if !scheduler.HasPlugin(targetTypeName) {
				return false, nil
			}
		}

		return true, nil
	})
	if err != nil {
		tl.Fatalf("Error starting for scheduler and plugins: %v", err)
	}
}

func enableTypeConfigResource(name, namespace string, config *restclient.Config, tl common.TestLogger) {
	for _, enableTypeDirective := range framework.LoadEnableTypeDirectives(tl) {
		resources, err := kfenable.GetResources(config, enableTypeDirective)
		if err != nil {
			tl.Fatalf("Error retrieving resource definitions for EnableTypeDirective %q: %v", enableTypeDirective.Name, err)
		}

		if enableTypeDirective.Name == name {
			err = kfenable.CreateResources(nil, config, resources, namespace, false)
			if err != nil {
				tl.Fatalf("Error creating resources for EnableTypeDirective %q: %v", enableTypeDirective.Name, err)
			}
		}
	}
}

func deleteTypeConfigResource(name, namespace string, config *restclient.Config, tl common.TestLogger) {
	qualifiedName := util.QualifiedName{Namespace: namespace, Name: name}
	err := kubefedctl.DisableFederation(nil, config, nil, qualifiedName, true, false, false)
	if err != nil {
		tl.Fatalf("Error disabling federation of target type %q: %v", qualifiedName, err)
	}
}
