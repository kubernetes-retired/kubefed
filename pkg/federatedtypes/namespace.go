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
	namespaceNamespaced  bool               = false
	namespaceAPIResource metav1.APIResource = metav1.APIResource{
		Name:       "namespaces",
		Group:      "",
		Kind:       "Namespace",
		Version:    "v1",
		Namespaced: namespaceNamespaced,
	}
	namespaceTypeConfig FederatedTypeConfig = FederatedTypeConfig{
		ComparisonType: util.ResourceVersion,
		Template: FederationAPIResource{
			APIResource: namespaceAPIResource,
			UseKubeAPI:  true,
		},
		Placement: FederationAPIResource{
			APIResource: apiResource("FederatedNamespacePlacement", "federatednamespaceplacements", namespaceNamespaced),
		},
		Target: namespaceAPIResource,
	}
)

func init() {
	RegisterFederatedTypeConfig(namespaceTypeConfig)
}

func IsNamespaceKind(kind string) bool {
	return kind == namespaceTypeConfig.Target.Kind
}
