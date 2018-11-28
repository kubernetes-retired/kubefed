/*
Copyright 2018 The Federation v2 Authors.

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
	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/schedulingtypes"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	"github.com/kubernetes-sigs/federation-v2/test/e2e/framework"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	restclient "k8s.io/client-go/rest"
)

func createTemplate(typeConfig typeconfig.Interface, kubeConfig *restclient.Config, namespace string) (string, error) {
	templateAPIResource := typeConfig.GetTemplate()
	templateClient, err := util.NewResourceClientFromConfig(kubeConfig, &templateAPIResource)
	if err != nil {
		return "", err
	}
	template, err := common.NewTestTemplate(typeConfig, namespace)
	if err != nil {
		return "", err
	}
	createdTemplate, err := templateClient.Resources(namespace).Create(template)
	if err != nil {
		return "", err
	}
	return createdTemplate.GetName(), nil
}

func waitForMatchingPlacement(tl common.TestLogger, typeConfig typeconfig.Interface, kubeConfig *restclient.Config, name, namespace string, expected schedulingtypes.ScheduleResult) error {
	placementAPIResource := typeConfig.GetPlacement()
	placementKind := placementAPIResource.Kind
	client, err := util.NewResourceClientFromConfig(kubeConfig, &placementAPIResource)
	if err != nil {
		return err
	}

	return wait.PollImmediate(framework.PollInterval, framework.TestContext.SingleCallTimeout, func() (bool, error) {
		placement, err := client.Resources(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				tl.Errorf("An error occurred while polling for %s %s/%s: %v", placementKind, namespace, name, err)
			}
			return false, nil
		}

		clusterNames, err := util.GetClusterNames(placement)
		if err != nil {
			tl.Errorf("An error occurred while retrieving cluster names for override %s %s/%s: %v", placementKind, namespace, name, err)
			return false, nil
		}
		return !expected.PlacementUpdateNeeded(clusterNames), nil
	})
}

func waitForMatchingOverride(tl common.TestLogger, typeConfig typeconfig.Interface, kubeConfig *restclient.Config, name, namespace string, expected schedulingtypes.ScheduleResult) error {
	overrideAPIResource := typeConfig.GetOverride()
	overrideKind := overrideAPIResource.Kind
	client, err := util.NewResourceClientFromConfig(kubeConfig, overrideAPIResource)
	if err != nil {
		return err
	}

	return wait.PollImmediate(framework.PollInterval, framework.TestContext.SingleCallTimeout, func() (bool, error) {
		override, err := client.Resources(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				tl.Errorf("An error occurred while polling for %s %s/%s: %v", overrideKind, namespace, name, err)
			}
			return false, nil
		}
		return !expected.OverrideUpdateNeeded(typeConfig, override), nil
	})
}
