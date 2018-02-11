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

package validation

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cluster-registry/pkg/apis/clusterregistry"
)

func TestValidateCluster(t *testing.T) {
	successCases := []clusterregistry.Cluster{
		{ObjectMeta: metav1.ObjectMeta{Name: "cluster-s"}},
	}
	for _, successCase := range successCases {
		errs := ValidateCluster(&successCase)
		if len(errs) != 0 {
			t.Errorf("expect success: %v", errs)
		}
	}

	errorCases := map[string]clusterregistry.Cluster{
		"invalid label": {
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster-f",
				Labels: map[string]string{
					"NoUppercaseOrSpecialCharsLike=Equals": "bar",
				},
			},
		},
		"invalid cluster name (is a subdomain)": {
			ObjectMeta: metav1.ObjectMeta{Name: "mycluster.mycompany"},
		},
		"clusterName is set": {
			ObjectMeta: metav1.ObjectMeta{Name: "mycluster", ClusterName: "nonEmpty"},
		},
	}
	for testName, errorCase := range errorCases {
		errs := ValidateCluster(&errorCase)
		if len(errs) == 0 {
			t.Errorf("expected failure for %s", testName)
		}
	}
}

func TestValidateClusterUpdate(t *testing.T) {
	type clusterUpdateTest struct {
		old    clusterregistry.Cluster
		update clusterregistry.Cluster
	}
	successCases := []clusterUpdateTest{
		{
			old: clusterregistry.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster-s"},
			},
			update: clusterregistry.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster-s"},
			},
		},
	}
	for _, successCase := range successCases {
		successCase.old.ObjectMeta.ResourceVersion = "1"
		successCase.update.ObjectMeta.ResourceVersion = "1"
		errs := ValidateClusterUpdate(&successCase.update, &successCase.old)
		if len(errs) != 0 {
			t.Errorf("expect success: %v", errs)
		}
	}

	errorCases := map[string]clusterUpdateTest{
		"cluster name changed": {
			old: clusterregistry.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster-s"},
			},
			update: clusterregistry.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster-newname"},
			},
		},
		"clusterName is set": {
			old: clusterregistry.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster-s"},
			},
			update: clusterregistry.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster-s", ClusterName: "nonEmpty"},
			},
		},
	}
	for testName, errorCase := range errorCases {
		errorCase.old.ObjectMeta.ResourceVersion = "1"
		errorCase.update.ObjectMeta.ResourceVersion = "1"
		errs := ValidateClusterUpdate(&errorCase.update, &errorCase.old)
		if len(errs) == 0 {
			t.Errorf("expected failure: %s", testName)
		}
	}
}
