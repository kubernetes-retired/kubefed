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

package kubefedcluster

import (
	"reflect"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/kubefed/pkg/apis/core/common"
	fedv1b1 "sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	"sigs.k8s.io/kubefed/pkg/controller/util"
)

func TestThresholdCheckedClusterStatus(t *testing.T) {

	epoch := metav1.Now()
	t1 := metav1.Time{Time: epoch.Add(1 * time.Second)}
	t2 := metav1.Time{Time: epoch.Add(2 * time.Second)}
	t3 := metav1.Time{Time: epoch.Add(3 * time.Second)}
	t4 := metav1.Time{Time: epoch.Add(4 * time.Second)}
	t5 := metav1.Time{Time: epoch.Add(5 * time.Second)}

	config := &util.ClusterHealthCheckConfig{
		Period:           10 * time.Second,
		FailureThreshold: 3,
		SuccessThreshold: 1,
		Timeout:          3 * time.Second,
	}

	testCases := map[string]struct {
		clusterStatus         *fedv1b1.KubeFedClusterStatus
		storedClusterData     *ClusterData
		expectedClusterStatus *fedv1b1.KubeFedClusterStatus
		expectedResultRun     int64
	}{
		"ClusterReadyAtBegining": {
			clusterStatus:         clusterStatus(corev1.ConditionTrue, t1, t1),
			storedClusterData:     &ClusterData{clusterStatus: nil, resultRun: 0},
			expectedClusterStatus: clusterStatus(corev1.ConditionTrue, t1, t1),
			expectedResultRun:     1,
		},
		"ClusterNotReadyAtBegining": {
			clusterStatus:         clusterStatus(corev1.ConditionFalse, t1, t1),
			storedClusterData:     &ClusterData{clusterStatus: nil, resultRun: 0},
			expectedClusterStatus: clusterStatus(corev1.ConditionFalse, t1, t1),
			expectedResultRun:     1,
		},
		"ClusterNotReadyButWithinFailureThreshold": {
			clusterStatus: clusterStatus(corev1.ConditionFalse, t3, t3),
			storedClusterData: &ClusterData{
				clusterStatus: clusterStatus(corev1.ConditionTrue, t2, t1),
				resultRun:     2},
			expectedClusterStatus: clusterStatus(corev1.ConditionTrue, t3, t1),
			expectedResultRun:     3,
		},
		"ClusterNotReadyAndCrossedFailureThreshold": {
			clusterStatus: clusterStatus(corev1.ConditionFalse, t4, t4),
			storedClusterData: &ClusterData{
				clusterStatus: clusterStatus(corev1.ConditionTrue, t3, t1),
				resultRun:     3},
			expectedClusterStatus: clusterStatus(corev1.ConditionFalse, t4, t4),
			expectedResultRun:     1,
		},
		"ClusterReturnToReadyState": {
			clusterStatus: clusterStatus(corev1.ConditionTrue, t5, t5),
			storedClusterData: &ClusterData{
				clusterStatus: clusterStatus(corev1.ConditionFalse, t4, t1),
				resultRun:     1},
			expectedClusterStatus: clusterStatus(corev1.ConditionTrue, t5, t5),
			expectedResultRun:     1,
		},
	}

	for testName, tc := range testCases {
		t.Run(testName, func(t *testing.T) {
			newClusterStatus := thresholdAdjustedClusterStatus(tc.clusterStatus, tc.storedClusterData, config)
			if !reflect.DeepEqual(tc.expectedClusterStatus, newClusterStatus) {
				t.Fatalf("Unexpected state, expected: %v, got:%v", tc.expectedClusterStatus, newClusterStatus)
			}

			if tc.expectedResultRun != tc.storedClusterData.resultRun {
				t.Fatalf("Unexpected resultRun, expected: %v, got:%v", tc.expectedResultRun, tc.storedClusterData.resultRun)
			}
		})
	}

}

func clusterStatus(status corev1.ConditionStatus, lastProbeTime, lastTransitionTime metav1.Time) *fedv1b1.KubeFedClusterStatus {
	return &fedv1b1.KubeFedClusterStatus{
		Conditions: []fedv1b1.ClusterCondition{{
			Type:               common.ClusterReady,
			Status:             status,
			LastProbeTime:      lastProbeTime,
			LastTransitionTime: &lastTransitionTime,
		}},
	}
}
