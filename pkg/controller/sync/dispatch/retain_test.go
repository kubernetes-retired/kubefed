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

package dispatch

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/pointer"

	"sigs.k8s.io/kubefed/pkg/controller/util"
)

func TestRetainClusterFields(t *testing.T) {
	testCases := map[string]struct {
		retainReplicas   bool
		desiredReplicas  int64
		clusterReplicas  int64
		expectedReplicas int64
	}{
		"replicas not retained when retainReplicas=false or is not present": {
			retainReplicas:   false,
			desiredReplicas:  1,
			clusterReplicas:  2,
			expectedReplicas: 1,
		},
		"replicas retained when retainReplicas=true": {
			retainReplicas:   true,
			desiredReplicas:  1,
			clusterReplicas:  2,
			expectedReplicas: 2,
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			desiredObj := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"replicas": testCase.desiredReplicas,
					},
				},
			}
			clusterObj := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"replicas": testCase.clusterReplicas,
					},
				},
			}
			fedObj := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"retainReplicas": testCase.retainReplicas,
					},
				},
			}
			if err := RetainClusterFields("", desiredObj, clusterObj, fedObj); err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			replicas, ok, err := unstructured.NestedInt64(desiredObj.Object, util.SpecField, util.ReplicasField)
			if !ok {
				t.Fatalf("Field 'spec.replicas' not found")
			}
			if err != nil {
				t.Fatalf("An unexpected error occurred")
			}
			if replicas != testCase.expectedReplicas {
				t.Fatalf("Expected %d replicas when retainReplicas=%v, got %d", testCase.expectedReplicas, testCase.retainReplicas, replicas)
			}
		})
	}
}

func TestRetainHealthCheckNodePortInServiceFields(t *testing.T) {
	tests := []struct {
		name          string
		desiredObj    *unstructured.Unstructured
		clusterObj    *unstructured.Unstructured
		retainSucceed bool
		expectedValue *int64
	}{
		{
			"cluster object has no healthCheckNodePort",
			&unstructured.Unstructured{
				Object: map[string]interface{}{},
			},
			&unstructured.Unstructured{
				Object: map[string]interface{}{},
			},
			true,
			nil,
		},
		{
			"cluster object has invalid healthCheckNodePort",
			&unstructured.Unstructured{
				Object: map[string]interface{}{},
			},
			&unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"healthCheckNodePort": "invalid string",
					},
				},
			},
			false,
			nil,
		},
		{
			"cluster object has healthCheckNodePort 0",
			&unstructured.Unstructured{
				Object: map[string]interface{}{},
			},
			&unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"healthCheckNodePort": int64(0),
					},
				},
			},
			true,
			nil,
		},
		{
			"cluster object has healthCheckNodePort 1000",
			&unstructured.Unstructured{
				Object: map[string]interface{}{},
			},
			&unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"healthCheckNodePort": int64(1000),
					},
				},
			},
			true,
			pointer.Int64Ptr(1000),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := retainServiceFields(test.desiredObj, test.clusterObj); (err == nil) != test.retainSucceed {
				t.Fatalf("test %s fails: unexpected returned error %v", test.name, err)
			}

			currentValue, ok, err := unstructured.NestedInt64(test.desiredObj.Object, "spec", "healthCheckNodePort")
			if err != nil {
				t.Fatalf("test %s fails: %v", test.name, err)
			}
			if !ok && test.expectedValue != nil {
				t.Fatalf("test %s fails: expect specified healthCheckNodePort but not found", test.name)
			}
			if ok && (test.expectedValue == nil || *test.expectedValue != currentValue) {
				t.Fatalf("test %s fails: unexpected current healthCheckNodePort %d", test.name, currentValue)
			}
		})
	}
}

func TestRetainCustomFields(t *testing.T) {
	fields := []interface{}{"spec.paused", "spec.replicas"}
	testCases := map[string]struct {
		retainCustomFields []interface{}
		desiredPaused      bool
		clusterPaused      bool
		expectedPaused     bool
		desiredReplicas    int64
		clusterReplicas    int64
		expectedReplicas   int64
	}{
		"replicas and pasued not retained when retainCustomFields=[]string{} or is not present": {
			retainCustomFields: []interface{}{},
			desiredReplicas:    1,
			clusterReplicas:    2,
			expectedReplicas:   1,
			desiredPaused:      true,
			clusterPaused:      false,
			expectedPaused:     true,
		},
		"replicas and pasued retained when retainReplicas=[]string{\"spec.paused\", \"spec.replicas\"}": {
			retainCustomFields: fields,
			desiredReplicas:    1,
			clusterReplicas:    2,
			expectedReplicas:   2,
			desiredPaused:      true,
			clusterPaused:      false,
			expectedPaused:     false,
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			desiredObj := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"replicas": testCase.desiredReplicas,
						"paused":   testCase.desiredPaused,
					},
				},
			}
			clusterObj := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"replicas": testCase.clusterReplicas,
						"paused":   testCase.clusterPaused,
					},
				},
			}
			fedObj := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"retainCustomFields": testCase.retainCustomFields,
					},
				},
			}
			if err := RetainClusterFields("", desiredObj, clusterObj, fedObj); err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			paused, ok, err := unstructured.NestedBool(desiredObj.Object, util.SpecField, "paused")
			if !ok {
				t.Fatalf("Field 'spec.paused' not found")
			}
			if err != nil {
				t.Fatalf("An unexpected error occurred")
			}
			if paused != testCase.expectedPaused {
				t.Fatalf("Expected %v paused when retainCustomFields=%v, got %v", testCase.expectedPaused, testCase.retainCustomFields, paused)
			}

			replicas, ok, err := unstructured.NestedInt64(desiredObj.Object, util.SpecField, util.ReplicasField)
			if !ok {
				t.Fatalf("Field 'spec.replicas' not found")
			}
			if err != nil {
				t.Fatalf("An unexpected error occurred")
			}
			if replicas != testCase.expectedReplicas {
				t.Fatalf("Expected %d replicas when retainCustomFields=%v, got %d", testCase.expectedReplicas, testCase.retainCustomFields, replicas)
			}
		})
	}
}
