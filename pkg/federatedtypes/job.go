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
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	jobNamespaced bool                = true
	jobTypeConfig FederatedTypeConfig = FederatedTypeConfig{
		ComparisonType: util.Generation,
		Template: FederationAPIResource{
			APIResource: apiResource("FederatedJob", "federatedjobs", jobNamespaced),
		},
		Placement: FederationAPIResource{
			APIResource: apiResource("FederatedJobPlacement", "federatedjobplacements", jobNamespaced),
		},
		Override: &FederationAPIResource{
			APIResource: apiResource("FederatedJobOverride", "federatedjoboverrides", jobNamespaced),
		},
		OverridePath: []string{"spec", "parallelism"},
		Target: metav1.APIResource{
			Name:       "jobs",
			Group:      "batch",
			Kind:       "Job",
			Version:    "v1",
			Namespaced: jobNamespaced,
		},
	}
)

func init() {
	RegisterFederatedTypeConfig(jobTypeConfig)
}
