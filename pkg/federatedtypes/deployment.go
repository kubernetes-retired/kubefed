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

package federatedtypes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	deploymentNamespaced bool                = true
	deploymentTypeConfig FederatedTypeConfig = FederatedTypeConfig{
		Template: FederationAPIResource{
			APIResource: apiResource("federatedDeployment", "federateddeployments", deploymentNamespaced),
		},
		Placement: FederationAPIResource{
			APIResource: apiResource("FederatedDeploymentPlacement", "federateddeploymentplacements", deploymentNamespaced),
		},
		Override: &FederationAPIResource{
			APIResource: apiResource("FederatedDeploymentOverride", "federateddeploymentoverrides", deploymentNamespaced),
		},
		OverridePath: []string{"spec", "replicas"},
		Target: metav1.APIResource{
			Name:       "deployments",
			Group:      "apps",
			Kind:       "Deployment",
			Version:    "v1",
			Namespaced: deploymentNamespaced,
		},
	}
)

func init() {
	RegisterFederatedTypeConfig(deploymentTypeConfig)
}
