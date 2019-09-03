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

package enable

import (
	"testing"

	fedv1b1 "sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
)

func TestIsEquivalentAPI(t *testing.T) {
	for k, gvs := range equivalentAPIs {
		baseAPI := fedv1b1.APIResource{
			PluralName: k,
			Group:      gvs[0].Group,
			Version:    gvs[0].Version,
		}

		for _, gv := range gvs[1:] {
			api := fedv1b1.APIResource{
				PluralName: k,
				Group:      gv.Group,
				Version:    gv.Version,
			}

			if !IsEquivalentAPI(&baseAPI, &api) {
				t.Fatalf("An unexpected error occurred: %q and %q should be equivalent.", baseAPI, api)
			}
		}
	}
}
